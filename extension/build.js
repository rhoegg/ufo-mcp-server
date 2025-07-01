#!/usr/bin/env node

/**
 * Build script for UFO MCP Desktop Extension
 * Creates a .dxt file that can be dragged into Claude Desktop
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

console.log('Building UFO MCP Desktop Extension...');

// Ensure we have the necessary files
const requiredFiles = ['manifest.json', 'package.json', 'index.js'];
for (const file of requiredFiles) {
    if (!fs.existsSync(file)) {
        console.error(`Error: Missing required file: ${file}`);
        process.exit(1);
    }
}

// Read and validate manifest
const manifest = JSON.parse(fs.readFileSync('manifest.json', 'utf8'));
console.log(`Building ${manifest.name} v${manifest.version}`);

// Create output directory
const outputDir = '../build';
if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
}

const outputFile = path.join(outputDir, 'ufo-mcp.dxt');

try {
    // Install dependencies if node_modules doesn't exist
    if (!fs.existsSync('node_modules')) {
        console.log('Installing dependencies...');
        execSync('npm install', { stdio: 'inherit' });
    }
    
    // Create the DXT package (it's essentially a zip file)
    console.log('Creating DXT package...');
    
    // For now, we'll create a simple zip structure
    // In a real implementation, you'd use the @anthropic-ai/dxt package
    const AdmZip = require('adm-zip');
    const zip = new AdmZip();
    
    // Add manifest
    zip.addFile('manifest.json', Buffer.from(JSON.stringify(manifest, null, 2)));
    
    // Add package.json
    const packageJson = JSON.parse(fs.readFileSync('package.json', 'utf8'));
    zip.addFile('package.json', Buffer.from(JSON.stringify(packageJson, null, 2)));
    
    // Add main script
    zip.addFile('index.js', fs.readFileSync('index.js'));
    
    // Add node_modules (only production dependencies)
    console.log('Bundling dependencies...');
    execSync('npm ci --only=production', { stdio: 'inherit' });
    
    function addDirectoryToZip(zip, dirPath, zipPath = '') {
        const items = fs.readdirSync(dirPath);
        for (const item of items) {
            const fullPath = path.join(dirPath, item);
            const zipItemPath = zipPath ? path.join(zipPath, item) : item;
            
            if (fs.statSync(fullPath).isDirectory()) {
                addDirectoryToZip(zip, fullPath, zipItemPath);
            } else {
                zip.addFile(zipItemPath, fs.readFileSync(fullPath));
            }
        }
    }
    
    if (fs.existsSync('node_modules')) {
        addDirectoryToZip(zip, 'node_modules', 'node_modules');
    }
    
    // Write the DXT file
    zip.writeZip(outputFile);
    
    console.log(`✅ Successfully built: ${outputFile}`);
    console.log(`📦 Extension size: ${(fs.statSync(outputFile).size / 1024 / 1024).toFixed(2)} MB`);
    console.log('');
    console.log('To install:');
    console.log('1. Open Claude Desktop');
    console.log('2. Go to Settings > Extensions');
    console.log(`3. Drag ${outputFile} into the extensions area`);
    console.log('');
    console.log('Before using:');
    console.log('Make sure the UFO MCP server is running:');
    console.log('docker run -d --name ufo-mcp-shared -p 8080:8080 ufo-mcp:local --transport http --port 8080');
    
} catch (error) {
    console.error('Build failed:', error.message);
    process.exit(1);
}