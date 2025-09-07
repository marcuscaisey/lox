import { ExtensionContext, LogOutputChannel, window, workspace } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from "vscode-languageclient/node";

const langServerEnabledKey = "lox.useLanguageServer";

let client: LanguageClient | undefined;

export function activate(context: ExtensionContext) {
  const logger = window.createOutputChannel("Lox", { log: true });
  context.subscriptions.push(logger);

  workspace.onDidChangeConfiguration((event) => {
    if (event.affectsConfiguration(langServerEnabledKey)) {
      onDidChangeUseLanguageServer(logger);
    }
  });

  onDidChangeUseLanguageServer(logger);
}

function onDidChangeUseLanguageServer(logger: LogOutputChannel) {
  const enabled = workspace.getConfiguration().get<boolean>(langServerEnabledKey, true);

  function logWithSetting(msg: string) {
    logger.info(`${msg} (${langServerEnabledKey}: ${enabled.toString()})`);
  }

  if (!enabled) {
    if (client) {
      logWithSetting(`Stopping language server loxls`);
      client.stop().then(
        () => {
          logger.info("Stopped language server loxls");
        },
        (reason: unknown) => {
          logger.error(`Failed to stop language server loxls: ${String(reason)}`);
        },
      );
      client = undefined;
    } else {
      logWithSetting("Not starting language server loxls");
    }
    return;
  }

  logWithSetting("Starting language server loxls");

  const serverOptions: ServerOptions = {
    command: "loxls",
    transport: TransportKind.stdio,
  };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ language: "lox" }],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher("*.lox"),
    },
  };
  client = new LanguageClient("lox", "loxls", serverOptions, clientOptions);

  client.start().then(
    () => {
      if (client) {
        const version = client.initializeResult?.serverInfo?.version ?? "";
        logger.info(`Started language server loxls (version: ${version})`);
      }
    },
    (reason: unknown) => {
      logger.error(`Failed to start language server loxls: ${String(reason)}`);
    },
  );
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
