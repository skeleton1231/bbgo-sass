import nextConfig from "eslint-config-next";

/** @type {import("eslint").Linter.Config[]} */
const config = [
  ...nextConfig,
  {
    files: ["**/*.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
    },
  },
];

export default config;
