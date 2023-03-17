module.exports = {
    extends: ['stylelint-config-standard', 'stylelint-config-prettier'],
    rules: {
        'no-invalid-position-at-import-rule': null,
        'at-rule-no-unknown': [
            true,
            {
                ignoreAtRules: ['tailwind', 'apply', 'layer', 'variants', 'responsive', 'screen']
            }
        ],
        'function-no-unknown': [
            true,
            {
                ignoreFunctions: ['theme', 'screen']
            }
        ],
        'value-keyword-case': [
            'lower',
            {
                ignoreFunctions: ['theme']
            }
        ],
        'selector-pseudo-class-no-unknown': [true, { ignorePseudoClasses: ['global'] }],
        'custom-property-empty-line-before': ['never']
    }
};
