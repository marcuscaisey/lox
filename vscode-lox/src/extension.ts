import * as fs from "node:fs/promises";
import * as path from "node:path";

import { ExtensionContext, LogOutputChannel, window, workspace } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from "vscode-languageclient/node";

const useLanguageServerKey = "lox.useLanguageServer";
const loxlsPathKey = "lox.loxlsPath";
const defaultLoxlsPath = "loxls";

let client: LanguageClient | undefined;
let logger: LogOutputChannel;

export async function activate(context: ExtensionContext): Promise<void> {
  logger = window.createOutputChannel("Lox", { log: true });
  context.subscriptions.push(logger);

  context.subscriptions.push(
    workspace.onDidChangeConfiguration(async (event) => {
      if (event.affectsConfiguration(useLanguageServerKey) || event.affectsConfiguration(loxlsPathKey)) {
        await onDidChangeLangServerConfig();
      }
    }),
  );

  await onDidChangeLangServerConfig();
}

async function onDidChangeLangServerConfig(): Promise<void> {
  const config = workspace.getConfiguration();
  const useLanguageServer = config.get<boolean>(useLanguageServerKey, true);
  const loxlsPath = config.get<string>(loxlsPathKey, defaultLoxlsPath);

  if (!useLanguageServer) {
    if (client) {
      logger.info(`Stopping language server loxls (${useLanguageServerKey}: false)`);
      await stopClient();
    } else {
      logger.info(`Not starting language server loxls (${useLanguageServerKey}: false)`);
    }
    return;
  }

  if (client) {
    logger.info(`Stopping language server loxls`);
    await stopClient();
  }

  if (!(await isExecutable(loxlsPath))) {
    let msg = `Cannot find the Lox language server "loxls"`;
    if (loxlsPath !== defaultLoxlsPath) {
      msg += `(${loxlsPathKey}: ${loxlsPath})`;
    }
    msg +=
      `. Check PATH, or install loxls and reload the window.` +
      ` Install loxls with "go install github.com/marcuscaisey/lox/loxls@latest".`;
    window.showErrorMessage(msg);
    return;
  }

  logger.info(`Starting language server loxls (${useLanguageServerKey}: true, ${loxlsPathKey}: "${loxlsPath}")`);

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

  try {
    await client.start();
  } catch (reason: unknown) {
    logger.error(`Failed to start language server loxls: ${String(reason)}`);
  }
  logger.info(`Started language server loxls (version: ${client.initializeResult?.serverInfo?.version ?? ""})`);
}

export async function isExecutable(nameOrPath: string): Promise<boolean> {
  if (path.isAbsolute(nameOrPath)) {
    if (await isFileExecutable(nameOrPath)) {
      return true;
    }
    return false;
  }

  const pathVar = process.env.PATH ?? "";
  for (const dir of pathVar.split(path.delimiter)) {
    const candidate = path.join(dir, nameOrPath);
    if (await isFileExecutable(candidate)) {
      return true;
    }
  }

  return false;
}

async function isFileExecutable(p: string): Promise<boolean> {
  try {
    const stats = await fs.stat(p);
    if (stats.isFile()) {
      await fs.access(p, fs.constants.F_OK | fs.constants.X_OK);
      return true;
    }
  } catch {
    return false;
  }
  return false;
}

async function stopClient(): Promise<void> {
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
  if (client) {
    return stopClient();
  }
}
