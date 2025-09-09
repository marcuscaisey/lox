import { ExtensionContext, LogOutputChannel, window, workspace } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from "vscode-languageclient/node";

const useLanguageServerKey = "lox.useLanguageServer";
const loxlsPathKey = "lox.loxlsPath";

let client: LanguageClient | undefined;

export async function activate(context: ExtensionContext): Promise<void> {
  const logger = window.createOutputChannel("Lox", { log: true });
  context.subscriptions.push(logger);

  context.subscriptions.push(
    workspace.onDidChangeConfiguration(async (event) => {
      if (event.affectsConfiguration(useLanguageServerKey) || event.affectsConfiguration(loxlsPathKey)) {
        await onDidChangeLangServerConfig(logger);
      }
    }),
  );

  await onDidChangeLangServerConfig(logger);
}

async function onDidChangeLangServerConfig(logger: LogOutputChannel): Promise<void> {
  const config = workspace.getConfiguration();
  const useLanguageServer = config.get<boolean>(useLanguageServerKey, true);
  const loxlsPath = config.get<string>(loxlsPathKey, "loxls");

  if (!useLanguageServer) {
    if (client) {
      logger.info(`Stopping language server loxls (${useLanguageServerKey}: false)`);
      await stopClient(logger);
    } else {
      logger.info(`Not starting language server loxls (${useLanguageServerKey}: false)`);
    }
    return;
  }

  if (client) {
    logger.info(`Stopping language server loxls`);
    await stopClient(logger);
  }

  // TODO: restart existing language server if still enabled
  logger.info(`Starting language server loxls (${useLanguageServerKey}: true, ${loxlsPathKey}: "${loxlsPath}")`);

  const serverOptions: ServerOptions = {
    // TODO: check whether this exists first
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

  try {
    await client.start();
  } catch (reason: unknown) {
    logger.error(`Failed to start language server loxls: ${String(reason)}`);
  }
  logger.info(`Started language server loxls (version: ${client.initializeResult?.serverInfo?.version ?? ""})`);
}

async function stopClient(logger: LogOutputChannel): Promise<void> {
  if (!client) {
    return;
  }
  try {
    await client.stop();
  } catch (reason: unknown) {
    logger.error(`Failed to stop language server loxls: ${String(reason)}`);
  }
  logger.info("Stopped language server loxls");
  client = undefined;
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
