import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;

export function activate(_context: vscode.ExtensionContext) {
  console.log("Activating Lox extension");

  const config = vscode.workspace.getConfiguration("lox");

  const useLanguageServer = config.get<boolean>("useLanguageServer", true);
  if (!useLanguageServer) {
    return;
  }

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
  console.log("Deactivating Lox extension");
  if (!client) {
    return undefined;
  }
  return client.stop();
}
