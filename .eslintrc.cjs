module.exports = {
    root: true,
    extends: ['standard', 'prettier'],
    globals: {},
    env: {
        node: true,
        es2020: true,
        amd: true,
        browser: true
    },
    parserOptions: {
        ecmaFeatures: {},
        ecmaVersion: 2020
    },
    settings: {},
    rules: {
        'no-console': 'off',
        'no-protoype-builtins': 'off'
    }
};
