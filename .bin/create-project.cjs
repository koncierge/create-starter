#!/usr/bin/env node

const { execSync } = require('child_process');

// Arguments
const projectName = process.argv[2];

// Commands
const checkout = `git clone --depth 1 https://github.com/koncierge/create-starter.git ${projectName}`;
const install = `cd ${projectName} && npm install`;

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
console.log(`1) Cloning repository under "${projectName}"`);
const checkedOut = run(checkout);
if (!checkedOut) process.exit(code:-1);

console.log(`2) Installing dependencies for "${projectName}"`);
const installedDeps = run(install);
if (!installedDeps) process.exit(code:-1);

console.log("Congratulations, you're ready to go!");
