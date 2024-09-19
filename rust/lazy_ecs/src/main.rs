use clap::Parser;

#[derive(Parser)]
#[command(name = "lazy-ecs")]
enum Cli {
    Ssh(SshArgs),
}

#[derive(clap::Args)]
struct SshArgs {
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

fn main() {
    let Cli::Ssh(args) = Cli::parse();
    println!("{:?}", args.cluster);
}
