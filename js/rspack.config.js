const path = require('path');

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
                use: {
                    loader: 'builtin:swc-loader',
                    options: {
                        jsc: {
                            parser: {
                                syntax: 'typescript'
                            }
                        }
                    }
                },
                type: 'javascript/auto'
            }
        ]
    },
    builtins: {
        copy: {
            patterns: [
                { from: 'node_modules/@xterm/xterm/lib/xterm.css', to: '.' }
            ]
        }
    },
    performance: {
        maxAssetSize: 2000000,
        maxEntrypointSize: 2000000
    }
};
