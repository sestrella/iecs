use aws_config::BehaviorVersion;
use aws_sdk_ecs::{
    error::SdkError,
    operation::{describe_clusters::DescribeClustersError, list_clusters::ListClustersError},
};
use clap::Parser;

#[derive(Parser)]
#[command(name = "lazy-ecs")]
enum Cli {
    Exec(ExecArgs),
}

#[derive(clap::Args)]
struct ExecArgs {
    #[arg(long)]
    cluster: Option<String>,
    #[arg(long)]
    task: Option<String>,
    #[arg(long)]
    container: Option<String>,
    #[arg(long, default_value = "/bin/sh")]
    command: String,
    #[arg(long, default_value_t = true)]
    interactive: bool,
}

#[derive(Debug)]
enum MyError {
    DescribeClustersError(SdkError<DescribeClustersError>),
    ListClustersError(SdkError<ListClustersError>),
}

impl From<SdkError<DescribeClustersError>> for MyError {
    fn from(value: SdkError<DescribeClustersError>) -> Self {
        Self::DescribeClustersError(value)
    }
}

impl From<SdkError<ListClustersError>> for MyError {
    fn from(value: SdkError<ListClustersError>) -> Self {
        Self::ListClustersError(value)
    }
}

#[tokio::main]
async fn main() -> Result<(), MyError> {
    let Cli::Exec(args) = Cli::parse();

    let config = aws_config::defaults(BehaviorVersion::latest()).load().await;
    let client = aws_sdk_ecs::Client::new(&config);
    let cluster = get_cluster(&client, &args.cluster).await?;
    println!("{:?}", cluster);
    Ok(())
}

async fn get_cluster(
    client: &aws_sdk_ecs::Client,
    cluster_name_option: &Option<String>,
) -> Result<aws_sdk_ecs::types::Cluster, MyError> {
    if let Some(cluster_name) = cluster_name_option {
        let output = client
            .describe_clusters()
            .clusters(cluster_name)
            .send()
            .await?;
        if let Some(clusters) = output.clusters {
            if let Some(cluster) = clusters.first() {
                // TODO: try to get rid of the clone
                return Ok(cluster.clone());
            }
        }
    } else {
        let _output = client.list_clusters().send().await?;
    }
    todo!()
}
