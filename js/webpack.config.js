const path = require('path');
const CopyPlugin = require('copy-webpack-plugin');

module.exports = {
    mode: 'production',
    entry: "./src/main.ts",
    output: {
        filename: "gotty-bundle.js",
        path: path.resolve(__dirname, 'dist'),
        publicPath: '/',
        clean: true
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
            }
        ]
    },
    plugins: [
        new CopyPlugin({
            patterns: [
                { from: 'node_modules/@xterm/xterm/lib/xterm.css', to: '.' }
            ]
        })
    ],
    performance: {
        maxAssetSize: 2000000,
        maxEntrypointSize: 2000000
    },
    optimization: {
        usedExports: true
    }
};
