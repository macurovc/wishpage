module.exports = {
    root: true,
    env: {
        es2020: true,
        node: true,
    },
    extends: [
        'eslint:recommended',
        'plugin:@typescript-eslint/recommended',
    ],
    parser: '@typescript-eslint/parser',
    parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        project: './tsconfig.json', // Point to your tsconfig.json
    },
    plugins: [
        '@typescript-eslint',
    ],
    rules: {
    },
};
