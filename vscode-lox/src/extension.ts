import * as fs from "node:fs";
import * as path from "node:path";

import { ExtensionContext, LogOutputChannel, window, workspace } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from "vscode-languageclient/node";

const useLanguageServerKey = "lox.useLanguageServer";
const loxlsPathKey = "lox.loxlsPath";

let client: LanguageClient | undefined;
let logger: LogOutputChannel;

export async function activate(context: ExtensionContext): Promise<void> {
  logger = window.createOutputChannel("Lox", { log: true });
  context.subscriptions.push(logger);

  context.subscriptions.push(
    workspace.onDidChangeConfiguration(async (event) => {
      if (event.affectsConfiguration(useLanguageServerKey) || event.affectsConfiguration(loxlsPathKey)) {
        await onDidChangeLangServerConfig(context);
      }
    }),
  );

  await onDidChangeLangServerConfig(context);
}

async function onDidChangeLangServerConfig(context: ExtensionContext): Promise<void> {
  const config = workspace.getConfiguration();
  const useLanguageServer = config.get<boolean>(useLanguageServerKey, true);
  if (!useLanguageServer) {
    if (client) {
      await ensureClientStopped(`${useLanguageServerKey}: false`);
    } else {
      logger.info(`Not starting language server (${useLanguageServerKey}: false)`);
    }
    return;
  }

  await ensureClientStopped();

  const loxlsPathValue = config.get<string>(loxlsPathKey, "");
  let loxlsPath: string;
  if (loxlsPathValue) {
    loxlsPath = loxlsPathValue;
    if (!(await isExecutable(loxlsPath))) {
      window.showErrorMessage(
        `Cannot find the Lox language server "loxls" (${loxlsPathKey}: ${loxlsPath}).` +
          ` Check PATH, or install loxls and reload the window.`,
      );
      return;
    }
  } else {
    let name = "loxls";
    switch (process.platform) {
      case "win32":
        name += ".exe";
        break;
      case "linux":
      case "darwin":
        break;
      default:
        window.showErrorMessage(
          `Pre-built Lox language server "loxls" not available on operating system "${process.platform}".` +
            ` To enable Lox language features, install "loxls" with "go install github.com/marcuscaisey/lox/loxls@latest" and then set "${loxlsPathKey}" to its path.` +
            ` To disable language features, set "${useLanguageServerKey}" to "false".`,
        );
        return;
    }
    loxlsPath = context.asAbsolutePath(path.join("out", name));
  }

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

  logger.info(`Starting language server (${useLanguageServerKey}: true, ${loxlsPathKey}: "${loxlsPathValue}")`);
  try {
    await client.start();
  } catch (reason: unknown) {
    logger.error(`Failed to start language server: ${String(reason)}`);
  }
  logger.info(`Started language server (version: ${client.initializeResult?.serverInfo?.version ?? ""})`);
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
    const stats = await fs.promises.stat(p);
    if (stats.isFile()) {
      await fs.promises.access(p, fs.constants.F_OK | fs.constants.X_OK);
      return true;
    }
  } catch {
    return false;
  }
  return false;
}

async function ensureClientStopped(context?: string): Promise<void> {
  if (!client) {
    return;
  }
  if (client.needsStop()) {
    let msg = "Stopping language server";
    if (context) {
      msg += ` (${context})`;
    }
    logger.info(msg);
    try {
      await client.stop();
    } catch (reason: unknown) {
      logger.error(`Failed to stop language server: ${String(reason)}`);
    }
    logger.info("Stopped language server");
  }
  client = undefined;
}

export function deactivate(): Thenable<void> | undefined {
  return ensureClientStopped("deactivating extension");
}
