const vscode = require('vscode');
const { LanguageClient } = require('vscode-languageclient/node');

let client;

function activate(context) {
    // Server options: launch `goks lsp`
    const serverOptions = {
        command: 'goks',
        args: ['lsp'],
    };

    // Client options: trigger for files with language 'gox'
    const clientOptions = {
        documentSelector: [{ scheme: 'file', language: 'gox' }],
        synchronize: {
            fileEvents: vscode.workspace.createFileSystemWatcher('**/*.gox'),
        },
    };

    client = new LanguageClient(
        'goksLSP',
        'GoKS GOX Language Server',
        serverOptions,
        clientOptions
    );

    client.start();
}

function deactivate() {
    if (!client) {
        return undefined;
    }
    return client.stop();
}

module.exports = {
    activate,
    deactivate,
};
