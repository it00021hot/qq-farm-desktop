import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(__dirname, '..');
const desktopRoot = path.resolve(frontendRoot, '..');
const vueRoot = path.resolve(desktopRoot, '..', 'vue-framework');
const distDir = path.join(frontendRoot, 'dist');
const vueDist = path.join(vueRoot, 'dist');

function run(cmd, args, cwd) {
  const r = spawnSync(cmd, args, { cwd, stdio: 'inherit', shell: process.platform === 'win32' });
  if (r.status !== 0) {
    process.exit(r.status ?? 1);
  }
}

function copyDir(src, dest) {
  fs.rmSync(dest, { recursive: true, force: true });
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.cpSync(src, dest, { recursive: true });
}

if (!fs.existsSync(vueRoot)) {
  console.error('vue-framework not found at', vueRoot);
  process.exit(1);
}

run('pnpm', ['run', 'build:desktop'], vueRoot);

if (!fs.existsSync(path.join(vueDist, 'index.html'))) {
  console.error('vue-framework build did not produce dist/index.html');
  process.exit(1);
}

copyDir(vueDist, distDir);
console.log('Copied vue-framework/dist → desktop/frontend/dist');
