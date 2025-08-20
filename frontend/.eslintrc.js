module.exports = {
    root: true,
    env: {
        browser: true,
        es2020: true,
        node: true, // Add node environment
    },
    extends: [
        'eslint:recommended',
        'plugin:@typescript-eslint/recommended',
        'plugin:react-hooks/recommended',
        'plugin:react/recommended', // Add react plugin
    ],
    parser: '@typescript-eslint/parser',
    parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        project: './tsconfig.json', // Point to your tsconfig.json
    },
    plugins: [
        '@typescript-eslint',
        'react', // Add react plugin
        'react-refresh'
    ],
    rules: {
        'react-refresh/only-export-components': 'warn',
        'react/react-in-jsx-scope': 'off', // Not needed with modern React
    },
    settings: {  // Add settings for the react plugin
        react: {
            version: 'detect',
        },
    },
};
