use std::fmt::Display;

use anyhow::Context;
use aws_config::BehaviorVersion;
use aws_sdk_ecs::types::Container;
use clap::Parser;
use inquire::Select;

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

struct SelectableContainer {
    name: String,
    arn: String,
}

impl TryFrom<Container> for SelectableContainer {
    type Error = &'static str;

    fn try_from(value: Container) -> Result<Self, Self::Error> {
        let name = value.name.ok_or("Name not found")?;
        let arn = value.container_arn.ok_or("ARN not found")?;
        Ok(SelectableContainer { name, arn })
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let Cli::Exec(args) = Cli::parse();

    let config = aws_config::defaults(BehaviorVersion::latest()).load().await;
    let client = aws_sdk_ecs::Client::new(&config);
    let cluster = get_cluster(&client, &args.cluster).await?;
    let task = get_task(&client, &cluster, &args.task).await?;
    let container = get_container(&client, &cluster, &task, &args.container).await?;
    println!("{:?}", container);
    Ok(())
}

// TODO: Improve error message
async fn get_cluster(
    client: &aws_sdk_ecs::Client,
    cluster_arg: &Option<String>,
) -> anyhow::Result<String> {
    if let Some(cluster_name) = cluster_arg {
        let output = client
            .describe_clusters()
            .clusters(cluster_name)
            .send()
            .await?;
        let clusters = output.clusters.context("")?;
        let cluster = clusters.first().context("")?;
        let cluster_arn = cluster.cluster_arn.as_ref().context("")?;
        return Ok(cluster_arn.to_string());
    }
    let output = client.list_clusters().send().await?;
    let clusters = output.cluster_arns.context("")?;
    let cluster = Select::new("Cluster", clusters).prompt().context("")?;
    Ok(cluster)
}

// TODO: take into account task_arg
async fn get_task(
    client: &aws_sdk_ecs::Client,
    cluster: &String,
    task_arg: &Option<String>,
) -> anyhow::Result<String> {
    let output = client.list_tasks().cluster(cluster).send().await?;
    let tasks = output.task_arns.context("")?;
    let task = Select::new("Task", tasks).prompt().context("")?;
    Ok(task)
}

async fn get_container(
    client: &aws_sdk_ecs::Client,
    cluster: &String,
    task: &String,
    container_arg: &Option<String>,
) -> anyhow::Result<String> {
    let output = client
        .describe_tasks()
        .cluster(cluster)
        .tasks(task)
        .send()
        .await?;
    let containers: Vec<Result<SelectableContainer, &'static str>> = output
        .tasks
        .unwrap_or(Vec::new())
        .into_iter()
        .map(|task| task.containers.unwrap_or(Vec::new()))
        .flatten()
        .map(|container| SelectableContainer::try_from(container))
        .collect();
    // let container = Select::new("Container", containers).prompt().context("")?;
    Ok("".to_string())
}
