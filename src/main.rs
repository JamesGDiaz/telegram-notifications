// src/main.rs

mod logging;

use actix_web::{web, App, HttpResponse, HttpServer, Responder};
use log::{error, info};
use serde::Deserialize;
use std::env;
use std::sync::Arc;
use teloxide::prelude::*;
use teloxide::types::ParseMode::MarkdownV2;
use tokio::sync::Mutex;
use tokio::time::{sleep, Duration};

#[derive(Deserialize)]
struct MessageInput {
    text: String,
    sender_id: Option<String>,
}

#[derive(Deserialize)]
struct SenderIdQuery {
    sender_id: Option<String>,
}

struct AppState {
    messages: Mutex<Vec<MessageInput>>,
    timer_running: Mutex<bool>,
}

async fn receive_message(
    data: web::Data<Arc<AppState>>,
    input: web::Json<MessageInput>,
    query: web::Query<SenderIdQuery>,
) -> impl Responder {
    // Append the received text to the messages vector
    let mut messages = data.messages.lock().await;
    messages.push(MessageInput {
        sender_id: query.sender_id.clone(),
        text: input.text.clone(),
    });

    // Check if the batching timer is running
    let mut timer_running = data.timer_running.lock().await;
    if !*timer_running {
        *timer_running = true;

        let data_clone = data.clone();

        // Start the batching timer (e.g., 60 seconds)
        tokio::spawn(async move {
            sleep(Duration::from_secs(5)).await;

            // After the timer expires, send the batched messages
            let mut messages = data_clone.messages.lock().await;
            if !messages.is_empty() {
                let message_text = messages
                    .iter()
                    .map(|message| {
                        if let Some(sender_id) = &message.sender_id {
                            format!("From {}: {}", sender_id, message.text)
                        } else {
                            message.text.clone()
                        }
                    })
                    .collect::<Vec<_>>()
                    .join("\n\n");
                send_telegram_message(message_text).await;
                messages.clear();
            }

            // Reset the timer flag
            let mut timer_running = data_clone.timer_running.lock().await;
            *timer_running = false;
        });
    }

    HttpResponse::Ok()
}

// Function to initialize the Telegram bot with the token from environment variables
fn get_bot() -> Bot {
    let bot_token = env::var("TELEGRAM_BOT_TOKEN").expect("TELEGRAM_BOT_TOKEN not set");
    Bot::new(bot_token)
}

// Function to get the chat ID from environment variables
fn get_chat_id() -> i64 {
    let chat_id_str = env::var("CHAT_ID").expect("CHAT_ID not set");
    chat_id_str.parse().expect("CHAT_ID must be an integer")
}

// Function to send the message to the Telegram user
async fn send_telegram_message(message_text: String) {
    let bot = get_bot();
    let chat_id = get_chat_id();

    let res = bot
        .send_message(ChatId(chat_id), message_text)
        .parse_mode(MarkdownV2)
        .send()
        .await;

    match res {
        Ok(_) => info!("New message sent"),
        Err(e) => error!("Error sending message: {:?}", e),
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let _ = dotenvy::dotenv();

    let _ = logging::init_logging();

    // Initialize the shared application state
    let app_state = Arc::new(AppState {
        messages: Mutex::new(Vec::new()),
        timer_running: Mutex::new(false),
    });

    // Start the Actix-web HTTP server
    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(app_state.clone()))
            .route("/notification", web::post().to(receive_message))
    })
    .bind(("127.0.0.1", 10000))? // Bind to port 10000
    .run()
    .await
}
