use std::fmt::Display;

use anyhow::{anyhow, ensure, Context};
use aws_config::BehaviorVersion;
use aws_sdk_ecs::types::{Cluster, Container};
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

struct SelectableCluster {
    name: String,
    arn: String,
}

impl Display for SelectableCluster {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} ({})", self.name, self.arn)
    }
}

impl TryFrom<&Cluster> for SelectableCluster {
    type Error = anyhow::Error;

    fn try_from(value: &Cluster) -> Result<Self, Self::Error> {
        let name = value
            .cluster_name
            .as_ref()
            .ok_or_else(|| anyhow!("cluster_name not found"))?;
        let arn = value
            .cluster_arn
            .as_ref()
            .ok_or_else(|| anyhow!("cluster_arn not found"))?;
        Ok(SelectableCluster {
            name: name.to_string(),
            arn: arn.to_string(),
        })
    }
}

impl TryFrom<String> for SelectableCluster {
    type Error = anyhow::Error;

    fn try_from(value: String) -> Result<Self, Self::Error> {
        let (_, name) = value.split_once("/").ok_or_else(|| anyhow!("TODO"))?;
        Ok(SelectableCluster {
            name: name.to_string(),
            arn: value,
        })
    }
}

struct SelectableTask {
    name: String,
    arn: String,
}

impl Display for SelectableTask {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} ({})", self.name, self.arn)
    }
}

impl TryFrom<String> for SelectableTask {
    type Error = anyhow::Error;

    fn try_from(value: String) -> Result<Self, Self::Error> {
        let (_, name) = value.split_once("/").ok_or_else(|| anyhow!("TODO"))?;
        Ok(SelectableTask {
            name: name.to_string(),
            arn: value,
        })
    }
}

struct SelectableContainer {
    name: String,
    arn: String,
    runtime_id: String,
}

impl Display for SelectableContainer {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} ({})", self.name, self.arn)
    }
}

impl TryFrom<Container> for SelectableContainer {
    type Error = anyhow::Error;

    fn try_from(value: Container) -> Result<Self, Self::Error> {
        let name = value.name.ok_or_else(|| anyhow!("name not found"))?;
        let arn = value
            .container_arn
            .ok_or_else(|| anyhow!("container_arn not found"))?;
        let runtime_id = value
            .runtime_id
            .ok_or_else(|| anyhow!("runtime_id not found"))?;
        Ok(SelectableContainer {
            name,
            arn,
            runtime_id,
        })
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let Cli::Exec(args) = Cli::parse();

    let config = aws_config::defaults(BehaviorVersion::latest()).load().await;
    let client = aws_sdk_ecs::Client::new(&config);
    let cluster = get_cluster(&client, &args.cluster).await?;
    let task = get_task(&client, &cluster.arn, &args.task).await?;
    let container = get_container(&client, &cluster.arn, &task.arn, &args.container).await?;
    println!("{}", container.name);
    Ok(())
}

async fn get_cluster(
    client: &aws_sdk_ecs::Client,
    cluster_arg: &Option<String>,
) -> anyhow::Result<SelectableCluster> {
    if let Some(cluster_name) = cluster_arg {
        let output = client
            .describe_clusters()
            .clusters(cluster_name)
            .send()
            .await?;
        let clusters = output.clusters.unwrap_or_else(|| Vec::new());
        let cluster = clusters
            .first()
            .with_context(|| format!("Cluster {} not found", cluster_name))?;
        return SelectableCluster::try_from(cluster);
    }
    let output = client.list_clusters().send().await?;
    let clusters = output
        .cluster_arns
        .unwrap_or_else(|| Vec::new())
        .into_iter()
        .map(|arn| SelectableCluster::try_from(arn))
        .collect::<anyhow::Result<Vec<SelectableCluster>>>()?;
    ensure!(!clusters.is_empty(), "No clusters found");
    let cluster = Select::new("Cluster", clusters)
        .prompt()
        .context("Error selecting cluster")?;
    Ok(cluster)
}

// TODO: take into account task_arg
async fn get_task(
    client: &aws_sdk_ecs::Client,
    cluster: &String,
    task_arg: &Option<String>,
) -> anyhow::Result<SelectableTask> {
    if let Some(task_name) = task_arg {
        let output = client.describe_tasks().tasks(task_name).send().await?;
        let tasks = output.tasks.unwrap_or_else(|| Vec::new());
        let task = tasks
            .first()
            .with_context(|| format!("Task {} not found", task_name));
        return SelectableTask::try_from(task);
    }
    let output = client.list_tasks().cluster(cluster).send().await?;
    let tasks = output
        .task_arns
        .unwrap_or_else(|| Vec::new())
        .into_iter()
        .map(|arn| SelectableTask::try_from(arn))
        .collect::<anyhow::Result<Vec<SelectableTask>>>()?;
    ensure!(!tasks.is_empty(), "No tasks found");
    let task = Select::new("Task", tasks).prompt().context("TODO")?;
    Ok(task)
}

async fn get_container(
    client: &aws_sdk_ecs::Client,
    cluster: &String,
    task: &String,
    container_arg: &Option<String>,
) -> anyhow::Result<SelectableContainer> {
    let output = client
        .describe_tasks()
        .cluster(cluster)
        .tasks(task)
        .send()
        .await?;
    let containers = output
        .tasks
        .unwrap_or(Vec::new())
        .into_iter()
        .map(|task| task.containers.unwrap_or(Vec::new()))
        .flatten()
        .map(|container| SelectableContainer::try_from(container))
        .collect::<anyhow::Result<Vec<SelectableContainer>>>()?;
    let container = Select::new("Container", containers)
        .prompt()
        .context("TODO")?;
    Ok(container)
}
