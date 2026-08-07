import { spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const vueRoot = path.resolve(__dirname, '..', '..', '..', 'qq-farm-web');

let port = process.env.WAILS_VITE_PORT || '9245';
const portIdx = process.argv.indexOf('--port');
if (portIdx >= 0 && process.argv[portIdx + 1]) {
  port = process.argv[portIdx + 1];
}

const child = spawn(
  'pnpm',
  ['exec', 'vite', '--mode', 'desktop', '--host', '127.0.0.1', '--port', String(port), '--strictPort'],
  { cwd: vueRoot, stdio: 'inherit', shell: process.platform === 'win32', env: process.env }
);

child.on('exit', (code) => process.exit(code ?? 0));
