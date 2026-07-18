import path from "node:path";

export function resolveAppPaths(config, options = {}) {
  const environment = options.env ?? process.env;
  const configDirectory = path.dirname(config.configPath);
  const appHome = path.resolve(
    options.home ?? environment.LEARNLOOM_HOME ?? configDirectory,
  );
  const dataDirectory = resolveFrom(appHome, config.storage.dataDirectory);
  const outputRoot = resolveFrom(appHome, config.storage.outputDirectory);
  const profileDataDirectory = path.join(dataDirectory, "profiles", config.profileId);
  const outputDirectory = path.join(outputRoot, config.profileId);

  return {
    appHome,
    dataDirectory,
    profileDataDirectory,
    outputRoot,
    outputDirectory,
    historyPath: path.join(profileDataDirectory, "history.json"),
    workspacePath: path.join(dataDirectory, "workspace.sqlite"),
    runsDirectory: path.join(profileDataDirectory, "runs"),
    locksDirectory: path.join(dataDirectory, "locks"),
    logsDirectory: path.join(dataDirectory, "logs"),
  };
}

function resolveFrom(base, value) {
  return path.isAbsolute(value) ? path.normalize(value) : path.resolve(base, value);
}
