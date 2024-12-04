// logging.rs

use log::LevelFilter;
use log4rs::{
    append::console::ConsoleAppender,
    config::{Appender, Config as LogConfig, Root},
    encode::pattern::PatternEncoder,
};
use std::error::Error;

pub fn init_logging() -> Result<(), Box<dyn Error>> {
    // Create console appender
    let stdout = ConsoleAppender::builder()
        .encoder(Box::new(PatternEncoder::new(
            "{d(%H:%M:%S | %d-%m-%Y)} {f}:{L} {h({l})}: {m}{n}",
        )))
        .build();

    let appenders = vec![Appender::builder().build("stdout", Box::new(stdout))];

    let root_builder = Root::builder().appender("stdout");

    let log_config = LogConfig::builder()
        .appenders(appenders)
        .build(root_builder.build(LevelFilter::Info))?;

    let _ = log4rs::init_config(log_config)?;

    Ok(())
}
