use std::{fmt::Display, process::Command};

use anyhow::{anyhow, ensure, Context};
use aws_config::BehaviorVersion;
use aws_sdk_ecs::types::{Cluster, Container, Session, Task};
use aws_sdk_ssm::operation::start_session::StartSessionInput;
use clap::Parser;
use inquire::Select;
use serde::{ser::SerializeStruct, Serialize};

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

impl TryFrom<&Task> for SelectableTask {
    type Error = anyhow::Error;

    fn try_from(value: &Task) -> Result<Self, Self::Error> {
        let arn = value.task_arn.as_ref().context("TODO")?;
        let (_, name) = arn.split_once("/").context("TODO")?;
        Ok(SelectableTask {
            name: name.to_string(),
            arn: arn.to_string(),
        })
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

struct SerializableSession(Session);

impl Serialize for SerializableSession {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let mut state = serializer.serialize_struct("SerializableSession", 3)?;
        state.serialize_field("SessionId", &self.0.session_id)?;
        state.serialize_field("StreamUrl", &self.0.stream_url)?;
        state.serialize_field("TokenValue", &self.0.token_value)?;
        state.end()
    }
}

struct SerializableStartSession(StartSessionInput);

impl Serialize for SerializableStartSession {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let mut state = serializer.serialize_struct("SerializableSession", 3)?;
        state.serialize_field("DocumentName", &self.0.document_name)?;
        state.serialize_field("Parameters", &self.0.parameters)?;
        state.serialize_field("Reason", &self.0.reason)?;
        state.serialize_field("Target", &self.0.target)?;
        state.end()
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

    let session = execute_command(
        &client,
        &cluster.arn,
        &task.arn,
        &container.name,
        &args.command,
        args.interactive,
    )
    .await?;

    let start_session = SerializableStartSession(
        StartSessionInput::builder()
            .target(format!(
                "ecs:{}_{}_{}",
                cluster.name, task.name, container.runtime_id
            ))
            .build()?,
    );

    let region = config
        .region()
        .ok_or_else(|| anyhow!("Region not available"))?;

    let mut command = Command::new("session-manager-plugin")
        .args([
            serde_json::to_string(&session)?,
            region.to_string(),
            "StartSession".to_string(),
            "".to_string(),
            serde_json::to_string(&start_session)?,
            format!("https://ssm.{}.amazonaws.com", region),
        ])
        .spawn()?;

    command.wait()?;

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
        let output = client
            .describe_tasks()
            .cluster(cluster)
            .tasks(task_name)
            .send()
            .await?;
        let tasks = output.tasks.unwrap_or_else(|| Vec::new());
        let task = tasks
            .first()
            .with_context(|| format!("Task {} not found", task_name))?;
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

// TODO: use container_arg
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

async fn execute_command(
    client: &aws_sdk_ecs::Client,
    cluster_arn: &String,
    task_arn: &String,
    container_name: &String,
    command: &String,
    interactive: bool,
) -> anyhow::Result<SerializableSession> {
    let output = client
        .execute_command()
        .cluster(cluster_arn)
        .task(task_arn)
        .container(container_name)
        .command(command)
        .interactive(interactive)
        .send()
        .await?;
    let session = output.session.ok_or(anyhow!("TODO"))?;
    Ok(SerializableSession(session))
}
