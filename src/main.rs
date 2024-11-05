use std::{fmt::Display, process::Command};

use anyhow::{ensure, Context};
use aws_config::BehaviorVersion;
use aws_sdk_ecs::types::{Cluster, Container, LogDriver, Session, Task};
use aws_sdk_ssm::operation::start_session::StartSessionInput;
use clap::Parser;
use inquire::Select;
use serde::{ser::SerializeStruct, Serialize};
use which::which;

static SESSION_MANAGER_PLUGIN_NOT_FOUND: &str = r#"
'session-manager-plugin' not found, install it using the instructions in the link below:

https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html
"#;

#[derive(Parser)]
#[command(name = "iecs")]
enum Cli {
    Exec(ExecArgs),
    Logs(LogsArgs),
}

#[derive(clap::Args)]
struct ExecArgs {
    #[arg(long)]
    cluster: Option<String>,
    #[arg(long)]
    task: Option<String>,
    #[arg(long)]
    container: Option<String>,
    #[arg(long, default_value = "/bin/bash")]
    command: String,
    #[arg(long, default_value_t = true)]
    interactive: bool,
}

#[derive(clap::Args)]
struct LogsArgs {
    #[arg(long)]
    cluster: Option<String>,
    #[arg(long)]
    task: Option<String>,
    #[arg(long)]
    container: Option<String>,
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
            .context("cluster_name not found")?;
        let arn = value
            .cluster_arn
            .as_ref()
            .context("cluster_arn not found")?;
        Ok(SelectableCluster {
            name: name.to_string(),
            arn: arn.to_string(),
        })
    }
}

impl TryFrom<String> for SelectableCluster {
    type Error = anyhow::Error;

    fn try_from(value: String) -> Result<Self, Self::Error> {
        let (_, name) = value
            .split_once("/")
            .with_context(|| format!("Unable to split {} by /", value))?;
        Ok(SelectableCluster {
            name: name.to_string(),
            arn: value,
        })
    }
}

struct SelectableTask(Task);

impl Display for SelectableTask {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let task_arn = self.0.task_arn.as_ref().unwrap();
        write!(f, "{}", task_arn)
    }
}

// impl TryFrom<Task> for SelectableTask {
//     type Error = anyhow::Error;
//
//     fn try_from(value: Task) -> Result<Self, Self::Error> {
//         let arn = value.task_arn.context("TODO")?;
//         let (_, name) = arn.split_once("/").context("TODO")?;
//         Ok(SelectableTask {
//             name: name.to_string(),
//             arn: arn.to_string(),
//             task_definition_arn: value.task_definition_arn,
//         })
//     }
// }
//
// impl TryFrom<&Task> for SelectableTask {
//     type Error = anyhow::Error;
//
//     fn try_from(value: &Task) -> Result<Self, Self::Error> {
//         let arn = value.task_arn.as_ref().context("TODO")?;
//         let (_, name) = arn.split_once("/").context("TODO")?;
//         Ok(SelectableTask {
//             name: name.to_string(),
//             arn: arn.to_string(),
//             task_definition_arn: value.task_definition_arn.clone(),
//         })
//     }
// }

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
        let name = value.name.context("'name' is not defined")?;
        let arn = value
            .container_arn
            .context("'container_arn' is not defined")?;
        let runtime_id = value.runtime_id.context("'runtime_id' is not defined")?;
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
    let config = aws_config::defaults(BehaviorVersion::latest()).load().await;
    let ecs_client = aws_sdk_ecs::Client::new(&config);
    match Cli::parse() {
        Cli::Exec(args) => run_exec(&ecs_client, &args).await,
        Cli::Logs(args) => {
            let cwlogs_client = aws_sdk_cloudwatchlogs::Client::new(&config);
            run_logs(&ecs_client, &cwlogs_client, &args).await
        }
    }
}

async fn run_exec(ecs_client: &aws_sdk_ecs::Client, args: &ExecArgs) -> anyhow::Result<()> {
    let session_manager_path =
        which("session-manager-plugin").context(SESSION_MANAGER_PLUGIN_NOT_FOUND)?;

    let cluster = get_cluster(&ecs_client, &args.cluster).await?;
    let task = get_task(&ecs_client, &cluster.arn, &args.task).await?;
    let task_arn = task.task_arn.context("TODO")?;
    let container = get_container(&ecs_client, &cluster.arn, &task_arn, &args.container).await?;

    let session = execute_command(
        &ecs_client,
        &cluster.arn,
        &task_arn,
        &container.name,
        &args.command,
        args.interactive,
    )
    .await?;

    let (_, task_name) = task_arn.split_once("/").context("TODO")?;
    let start_session = SerializableStartSession(
        StartSessionInput::builder()
            .target(format!(
                "ecs:{}_{}_{}",
                cluster.name, task_name, container.runtime_id
            ))
            .build()?,
    );

    let region = ecs_client
        .config()
        .region()
        .context("'region' is not defined")?;

    let mut command = Command::new(session_manager_path)
        .args([
            serde_json::to_string(&session)?,
            region.to_string(),
            "StartSession".to_string(),
            // TODO: pass profile
            "".to_string(),
            serde_json::to_string(&start_session)?,
            format!("https://ssm.{}.amazonaws.com", region),
        ])
        .spawn()?;

    command.wait()?;

    Ok(())
}

async fn run_logs(
    ecs_client: &aws_sdk_ecs::Client,
    _cwlogs_client: &aws_sdk_cloudwatchlogs::Client,
    args: &LogsArgs,
) -> anyhow::Result<()> {
    let cluster = get_cluster(&ecs_client, &args.cluster).await?;
    let task = get_task(&ecs_client, &cluster.arn, &args.task).await?;
    let task_arn = task.task_arn.context("TODO")?;
    let container = get_container(&ecs_client, &cluster.arn, &task_arn, &args.container).await?;
    let task_definition_arn = task
        .task_definition_arn
        .context("'task_definition_arn' is not defined")?;
    let output = ecs_client
        .describe_task_definition()
        .task_definition(task_definition_arn)
        .send()
        .await?;
    let container_definition = output
        .task_definition
        .context("'task_definition' is not defined")?
        .container_definitions
        .context("'container_definitions' is not defined")?
        .into_iter()
        .find(|container_definition| container_definition.name.as_ref() == Some(&container.name))
        .context("TODO")?;
    let log_configuration = container_definition.log_configuration.context("TODO")?;
    let log_driver = log_configuration.log_driver;
    ensure!(
        log_driver != LogDriver::Awslogs,
        "Unsupported log driver '{}'",
        log_driver
    );
    let log_options = log_configuration.options.context("TODO")?;
    let _log_group = log_options.get("awslogs-group").context("TODO")?;
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
        let clusters = output.clusters.context("clusters is not defined")?;
        let cluster = clusters
            .first()
            .with_context(|| format!("cluster '{}' not found", cluster_name))?;
        return SelectableCluster::try_from(cluster);
    }
    let output = client.list_clusters().send().await?;
    let clusters = output
        .cluster_arns
        .unwrap_or_else(|| Vec::new())
        .into_iter()
        .map(|arn| SelectableCluster::try_from(arn))
        .collect::<anyhow::Result<Vec<SelectableCluster>>>()?;
    ensure!(!clusters.is_empty(), "no clusters found");
    let cluster = Select::new("Cluster", clusters)
        .prompt()
        .context("unable to render cluster selector")?;
    Ok(cluster)
}

async fn get_task(
    client: &aws_sdk_ecs::Client,
    cluster: &String,
    task_arg: &Option<String>,
) -> anyhow::Result<Task> {
    if let Some(task_name) = task_arg {
        let output = client
            .describe_tasks()
            .cluster(cluster)
            .tasks(task_name)
            .send()
            .await?;
        let tasks = output.tasks.context("tasks is not defined")?;
        let task = tasks
            .first()
            .with_context(|| format!("task '{}' not found", task_name))?;
        Ok(task.clone())
    } else {
        let task_arns = client.list_tasks().cluster(cluster).send().await?.task_arns;
        let output = client
            .describe_tasks()
            .cluster(cluster)
            .set_tasks(task_arns)
            .send()
            .await?;
        let tasks = output
            .tasks
            .unwrap_or_else(|| Vec::new())
            .into_iter()
            .map(|task| SelectableTask(task))
            .collect::<Vec<SelectableTask>>();
        ensure!(!tasks.is_empty(), "no tasks found");
        let SelectableTask(task) = Select::new("Task", tasks)
            .prompt()
            .context("unable to render task selector")?;
        Ok(task)
    }
}

async fn get_container(
    client: &aws_sdk_ecs::Client,
    cluster_id: &String,
    task_id: &String,
    container_arg: &Option<String>,
) -> anyhow::Result<SelectableContainer> {
    let output = client
        .describe_tasks()
        .cluster(cluster_id)
        .tasks(task_id)
        .send()
        .await?;
    let containers = output
        .tasks
        .unwrap_or(Vec::new())
        .into_iter()
        .map(|task| task.containers.unwrap_or_else(|| Vec::new()))
        .flatten()
        .map(|container| SelectableContainer::try_from(container))
        .collect::<anyhow::Result<Vec<SelectableContainer>>>()?;
    let container = if let Some(container_id) = container_arg {
        containers
            .into_iter()
            .find(|container| container.name == *container_id)
            .with_context(|| format!("container '{}' not found", container_id))?
    } else {
        Select::new("Container", containers)
            .prompt()
            .context("unable to render container selector")?
    };
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
    let session = output.session.context("session not found")?;
    Ok(SerializableSession(session))
}
