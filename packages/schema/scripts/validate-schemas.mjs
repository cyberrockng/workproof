import { readFile } from "node:fs/promises";
import { readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const schemaDir = fileURLToPath(new URL("../schemas/", import.meta.url));
const files = readdirSync(schemaDir).filter((file) => file.endsWith(".schema.json"));

if (files.length === 0) {
  throw new Error("no schemas found");
}

for (const file of files) {
  const path = join(schemaDir, file);
  const schema = JSON.parse(await readFile(path, "utf8"));
  for (const key of ["$schema", "$id", "title", "type"]) {
    if (!schema[key]) {
      throw new Error(`${file} is missing ${key}`);
    }
  }
}

console.log(`validated ${files.length} schema files`);
