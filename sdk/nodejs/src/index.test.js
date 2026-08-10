import { test } from 'node:test';
import assert from 'node:assert';
import { ProxyMeshPeer } from './index.js';

test('ProxyMeshPeer initializes with defaults', () => {
  const peer = new ProxyMeshPeer();
  assert.strictEqual(peer.baseURL, 'http://localhost:8000/api/peer');
  assert.strictEqual(peer.token, null);
  assert.strictEqual(peer.nodeId, null);
});

test('ProxyMeshPeer accepts custom options', () => {
  const peer = new ProxyMeshPeer({
    baseURL: 'https://api.example.com/api/peer',
    token: 'test-token',
    nodeId: 'node-1',
  });
  assert.strictEqual(peer.baseURL, 'https://api.example.com/api/peer');
  assert.strictEqual(peer.token, 'test-token');
  assert.strictEqual(peer.nodeId, 'node-1');
});

test('setAuth and clearAuth work correctly', () => {
  const peer = new ProxyMeshPeer();
  peer.setAuth('my-token', 'my-node');
  assert.strictEqual(peer.token, 'my-token');
  assert.strictEqual(peer.nodeId, 'my-node');
  peer.clearAuth();
  assert.strictEqual(peer.token, null);
  assert.strictEqual(peer.nodeId, null);
});
