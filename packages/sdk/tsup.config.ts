import { defineConfig } from "tsup";
import { baseTsupConfig } from "../../configs/tsup/base"; 

export default defineConfig({
  ...baseTsupConfig,
  entry: ["src/index.ts"], 
});