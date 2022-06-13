#!/usr/bin/env node

const { execSync } = require('child_process');

// Arguments
const projectName = process.argv[2];

// Commands
const checkout = `git clone --depth 1 https://github.com/koncierge/create-starter.git ${projectName}`;

const run = (command) => {
    try {
        execSync(`${command}`, { stdio: 'inherit' });
    } catch (e) {
        console.error(`Failed to execute ${command}`, e);
        return false;
    }
    return true;
};

// Execute
const checkedOut = run(checkout);
if (!checkedOut) process.exit(-1);

console.log('\n@koncierge/create-starter\n');
console.log("Congratulations, you're ready to go!");
console.log(`Run the following commands to install dependencies:\n\ncd ${projectName} && yarn\n`);
