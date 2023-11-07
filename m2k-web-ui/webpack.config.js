const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const CopyPlugin = require("copy-webpack-plugin");

module.exports = {
    mode: 'development',
    entry: './src/index.js',
    plugins: [
        new HtmlWebpackPlugin({
            title: 'Move2Kube in WASM',
        }),
        new CopyPlugin({
            patterns: [
                { from: path.resolve(__dirname, '..', 'bin', 'move2kube.wasm'), to: "move2kube.wasm" },
            ],
        }),
    ],
    output: {
        clean: true,
        filename: '[name].bundle.js',
        path: path.resolve(__dirname, 'dist'),
    },
    module: {
        rules: [
            {
                test: /\.css$/i,
                use: ['style-loader', 'css-loader'],
            },
        ],
    },
};