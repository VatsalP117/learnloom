import path from "node:path";

export function resolveAppPaths(config, options = {}) {
  const environment = options.env ?? process.env;
  const configDirectory = path.dirname(config.configPath);
  const appHome = path.resolve(
    options.home ?? environment.LEARNLOOM_HOME ?? configDirectory,
  );
  const dataDirectory = resolveFrom(appHome, config.storage.dataDirectory);
  const outputDirectory = resolveFrom(appHome, config.storage.outputDirectory);

  return {
    appHome,
    dataDirectory,
    outputDirectory,
    historyPath: path.join(dataDirectory, "history.json"),
    runsDirectory: path.join(dataDirectory, "runs"),
    locksDirectory: path.join(dataDirectory, "locks"),
    logsDirectory: path.join(dataDirectory, "logs"),
  };
}

function resolveFrom(base, value) {
  return path.isAbsolute(value) ? path.normalize(value) : path.resolve(base, value);
}
