import { ExtensionContext, LogOutputChannel, window, workspace } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from "vscode-languageclient/node";

const useLanguageServerKey = "lox.useLanguageServer";
const loxlsPathKey = "lox.loxlsPath";

let client: LanguageClient | undefined;

export function activate(context: ExtensionContext): void {
  const logger = window.createOutputChannel("Lox", { log: true });
  context.subscriptions.push(logger);

  context.subscriptions.push(
    workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration(useLanguageServerKey) || event.affectsConfiguration(loxlsPathKey)) {
        onDidChangeLangServerConfig(logger);
      }
    }),
  );

  onDidChangeLangServerConfig(logger);
}

function onDidChangeLangServerConfig(logger: LogOutputChannel): void {
  const config = workspace.getConfiguration();
  const useLanguageServer = config.get<boolean>(useLanguageServerKey, true);
  const loxlsPath = config.get<string>(loxlsPathKey, "loxls");

  if (!useLanguageServer) {
    if (client) {
      logger.info(`Stopping language server loxls (${useLanguageServerKey}: false)`);
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
      logger.info(`Not starting language server loxls (${useLanguageServerKey}: false)`);
    }
    return;
  }

  logger.info(
    `Starting language server loxls (${useLanguageServerKey}: true, ${loxlsPathKey}: "${loxlsPath}")`,
  );

  const serverOptions: ServerOptions = {
    command: loxlsPath,
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
