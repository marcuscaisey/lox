import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

const useLanguageServerKey = "lox.useLanguageServer";

export function activate(context: vscode.ExtensionContext) {
  const logger = vscode.window.createOutputChannel("Lox", { log: true });
  context.subscriptions.push(logger);

  const config = vscode.workspace.getConfiguration();

  const useLanguageServer = config.get<boolean>(useLanguageServerKey, true);
  if (!useLanguageServer) {
    logger.info(
      `Not starting language server loxls (${useLanguageServerKey}: ${useLanguageServer.toString()})`,
    );
    return;
  }
  logger.info(
    `Starting language server loxls (${useLanguageServerKey}: ${useLanguageServer.toString()})`,
  );

  const serverOptions: ServerOptions = {
    command: "loxls",
    transport: TransportKind.stdio,
  };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ language: "lox" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("*.lox"),
    },
  };
  client = new LanguageClient("lox", "loxls", serverOptions, clientOptions);
  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
