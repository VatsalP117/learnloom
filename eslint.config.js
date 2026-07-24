import js from "@eslint/js";
import react from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import globals from "globals";
import tseslint from "typescript-eslint";

export default [
  { ignores: ["web/dist/**"] },
  {
    files: ["web/src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: "latest",
      globals: globals.browser,
      parser: tseslint.parser,
      parserOptions: { ecmaFeatures: { jsx: true }, sourceType: "module" },
    },
    plugins: {
      "@typescript-eslint": tseslint.plugin,
      react,
      "react-hooks": reactHooks,
    },
    rules: {
      ...js.configs.recommended.rules,
      ...tseslint.configs.recommended.rules,
      ...reactHooks.configs.flat.recommended.rules,
      "react/jsx-uses-vars": "error",
      "react-hooks/set-state-in-effect": "off",
      "no-undef": "off",
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
    },
  },
];
