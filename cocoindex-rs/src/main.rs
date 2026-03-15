use clap::{Parser, Subcommand};
use coco_rs::{Indexer, Store, Provider, config::{Config, UserSettings, ProjectSettings}};
use std::path::PathBuf;
use std::sync::Arc;

#[derive(Parser)]
#[command(name = "coco-rs")]
#[command(about = "CocoIndex-Code Rust implementation", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,

    #[arg(short, long, env = "OPENAI_API_KEY")]
    api_key: Option<String>,

    #[arg(short, long, env = "OPENAI_API_BASE")]
    api_base: Option<String>,

    #[arg(short, long, env = "EMBEDDING_MODEL")]
    model: Option<String>,
}

#[derive(Subcommand)]
enum Commands {
    /// Index a project directory
    Index {
        #[arg(value_name = "PATH", default_value = ".")]
        path: PathBuf,
    },
    /// Search code in the index
    Search {
        #[arg(value_name = "QUERY")]
        query: String,
        #[arg(long, default_value = "5")]
        limit: usize,
        #[arg(long, default_value = "0")]
        offset: usize,
        #[arg(long)]
        languages: Option<Vec<String>>,
        #[arg(long)]
        paths: Option<Vec<String>>,
    },
    /// Start as MCP server
    Mcp,
    /// Initialize project settings
    Init {
        #[arg(value_name = "PATH", default_value = ".")]
        path: PathBuf,
    },
    /// Show project status
    Status,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();

    let cli = Cli::parse();

    // Disable logging for MCP mode to not corrupt stdout
    if !matches!(cli.command, Some(Commands::Mcp)) {
        tracing_subscriber::fmt::init();
    }

    // Load user settings
    let user_settings = UserSettings::load_or_default();

    // Build config from user settings and CLI args
    let mut config = Config {
        api_key: cli.api_key.unwrap_or(user_settings.api_key),
        api_base: cli.api_base.unwrap_or(user_settings.api_base),
        model: cli.model.unwrap_or(user_settings.model),
        embedding_dim: user_settings.embedding_dim,
        db_path: ".cocoindex_code/target_sqlite.db".to_string(),
    };

    match cli.command {
        Some(Commands::Init { path }) => {
            let settings = ProjectSettings::default();
            settings.save(&path)?;
            println!("Initialized project settings at {}/.cocoindex_code/settings.yml", path.display());

            // Also create user settings if they don't exist
            if UserSettings::load().is_err() {
                let user_settings = UserSettings::default();
                user_settings.save()?;
                println!("Created user settings at ~/.cocoindex_code/settings.yml");
                println!("Please update with your API key.");
            }
        }
        Some(Commands::Status) => {
            let path = std::env::current_dir()?;
            let db_path = path.join(&config.db_path);

            if db_path.exists() {
                println!("Database: {}", db_path.display());
                println!("Model: {}", config.model);
                println!("API Base: {}", config.api_base);
            } else {
                println!("No index found. Run 'coco-rs index' to create one.");
            }
        }
        Some(Commands::Index { path }) => {
            let abs_path = path.canonicalize()?;
            config.db_path = abs_path.join(&config.db_path).to_string_lossy().to_string();

            let store = Arc::new(Store::new(&config).await?);
            let provider = Arc::new(Provider::new(&config));
            let indexer = Indexer::new(store.clone_internal(), provider.clone_internal(), &abs_path)?;

            indexer.index_directory(&abs_path).await?;
            println!("Indexing complete.");
        }
        Some(Commands::Search { query, limit, offset, languages, paths }) => {
            let path = std::env::current_dir()?;
            config.db_path = path.join(&config.db_path).to_string_lossy().to_string();

            let store = Arc::new(Store::new(&config).await?);
            let provider = Arc::new(Provider::new(&config));

            let embeddings = provider.get_embeddings(vec![query]).await?;
            let embedding = embeddings.into_iter().next()
                .ok_or_else(|| anyhow::anyhow!("API returned empty embeddings"))?;

            let results = store.search(
                &embedding,
                limit,
                offset,
                languages.as_deref(),
                paths.as_deref(),
            ).await?;

            for (i, result) in results.iter().enumerate() {
                println!("{}. {} (Lines: {}-{}, Score: {:.4})",
                    i + 1, result.file_path, result.start_line, result.end_line, result.score);
                println!("---\n{}\n---", result.content);
            }
        }
        Some(Commands::Mcp) => {
            let path = std::env::current_dir()?;
            config.db_path = path.join(&config.db_path).to_string_lossy().to_string();

            let store = Arc::new(Store::new(&config).await?);
            let provider = Arc::new(Provider::new(&config));

            coco_rs::mcp::run(store, provider).await?;
        }
        None => {
            println!("No command provided. Use --help for usage.");
        }
    }

    Ok(())
}
