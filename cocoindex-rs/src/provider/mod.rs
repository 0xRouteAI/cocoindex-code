use serde::{Deserialize, Serialize};
use crate::config::Config;

pub struct Provider {
    api_key: String,
    api_base: String,
    model: String,
    client: reqwest::Client,
}

#[derive(Serialize)]
struct EmbeddingRequest {
    input: Vec<String>,
    model: String,
}

#[derive(Deserialize)]
struct EmbeddingResponse {
    data: Vec<EmbeddingData>,
}

#[derive(Deserialize)]
struct EmbeddingData {
    embedding: Vec<f32>,
}

impl Provider {
    pub fn new(config: &Config) -> Self {
        Self {
            api_key: config.api_key.clone(),
            api_base: config.api_base.clone(),
            model: config.model.clone(),
            client: reqwest::Client::new(),
        }
    }

    pub async fn get_embeddings(&self, texts: Vec<String>) -> anyhow::Result<Vec<Vec<f32>>> {
        if texts.is_empty() {
            return Ok(Vec::new());
        }

        let url = format!("{}/embeddings", self.api_base.trim_end_matches('/'));
        let res = self.client.post(url)
            .bearer_auth(&self.api_key)
            .json(&EmbeddingRequest {
                input: texts,
                model: self.model.clone(),
            })
            .send()
            .await?
            .error_for_status()?;

        let body: EmbeddingResponse = res.json().await?;

        if body.data.is_empty() {
            anyhow::bail!("API returned empty embeddings");
        }

        Ok(body.data.into_iter().map(|d| d.embedding).collect())
    }

    pub fn clone_internal(&self) -> Self {
        Self {
            api_key: self.api_key.clone(),
            api_base: self.api_base.clone(),
            model: self.model.clone(),
            client: self.client.clone(),
        }
    }
}
