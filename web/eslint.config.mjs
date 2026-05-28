import nextConfig from "eslint-config-next";

/** @type {import("eslint").Linter.Config[]} */
export default [
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
