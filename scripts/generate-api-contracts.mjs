import { readFile, writeFile } from "node:fs/promises";
import process from "node:process";

const serverPath = new URL("../internal/httpapp/response.go", import.meta.url);
const outputPath = new URL("../web/src/api-contracts.generated.ts", import.meta.url);
const server = await readFile(serverPath, "utf8");
const registry = server.match(/var knownProblemCodes = map\[string\]bool\{([\s\S]*?)\n\}/);
if (!registry) throw new Error("knownProblemCodes registry was not found");
const codes = [...registry[1].matchAll(/"([a-z_]+)"\s*:\s*true/g)]
  .map((match) => match[1])
  .sort();
if (codes.length === 0) throw new Error("knownProblemCodes registry is empty");

const generated = `// Code generated from the server problem contract. DO NOT EDIT.

export const apiProblemCodes = [
${codes.map((code) => `  "${code}",`).join("\n")}
] as const;

export type APIProblemCode = (typeof apiProblemCodes)[number];

export interface APIProblem {
  code: APIProblemCode;
  message: string;
}

export function isAPIProblemCode(value: unknown): value is APIProblemCode {
  return typeof value === "string" &&
    (apiProblemCodes as readonly string[]).includes(value);
}
`;

if (process.argv.includes("--check")) {
  const current = await readFile(outputPath, "utf8").catch(() => "");
  if (current !== generated) {
    process.stderr.write(
      "Generated API contracts are stale. Run npm run api:generate.\\n",
    );
    process.exitCode = 1;
  }
} else {
  await writeFile(outputPath, generated);
}
