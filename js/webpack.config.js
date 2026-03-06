const UglifyJSPlugin = require('uglifyjs-webpack-plugin');
const path = require('path');
const fs = require('fs');

// Custom plugin to copy WASM file
class CopyWasmPlugin {
    apply(compiler) {
        compiler.plugin('after-emit', (compilation, callback) => {
            const sourcePath = path.resolve(__dirname, 'node_modules/@bitwarden/sdk-internal/bitwarden_wasm_internal_bg.wasm');
            const destPath = path.resolve(__dirname, 'dist/bitwarden_wasm_internal_bg.wasm');

            fs.copyFileSync(sourcePath, destPath);
            console.log('Copied WASM file to dist');

            callback();
        });
    }
}

module.exports = {
    entry: "./src/main.ts",
    output: {
        filename: "./dist/gotty-bundle.js",
        publicPath: '/'
    },
    devtool: "source-map",
    resolve: {
        extensions: [".ts", ".tsx", ".js"]
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                loader: "ts-loader",
                exclude: /node_modules/
            },
            {
                test: /\.js$/,
                include: /node_modules/,
                loader: 'license-loader'
            }
        ]
    },
    plugins: [
        new UglifyJSPlugin(),
        new CopyWasmPlugin()
    ]
};
