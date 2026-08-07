import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { api, LoginRequest } from '../services/api';
import { Colors, Spacing, Typography, Layout } from '../theme';

interface Props {
  onLogin: () => void;
}

export default function LoginScreen({ onLogin }: Props) {
  const [nodeId, setNodeId] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleLogin = async () => {
    if (!nodeId.trim()) {
      setError('Enter your Node ID');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await api.login(nodeId.trim());
      api.setAuth(response.token, response.node_id);
      onLogin();
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.card}>
        <Text style={styles.title}>ProxyMesh Peer</Text>
        <Text style={styles.subtitle}>Connect your residential node to the network</Text>

        <TextInput
          style={styles.input}
          placeholder="Node ID"
          placeholderTextColor={Colors.textMuted}
          value={nodeId}
          onChangeText={setNodeId}
          autoCapitalize="none"
          autoCorrect={false}
        />

        {error ? <Text style={styles.errorText}>{error}</Text> : null}

        <TouchableOpacity
          style={[styles.button, loading && styles.buttonDisabled]}
          onPress={handleLogin}
          disabled={loading}
        >
          {loading ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Connect Node</Text>
          )}
        </TouchableOpacity>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.bg,
    justifyContent: 'center',
    alignItems: 'center',
    padding: Layout.screenPadding,
  },
  card: {
    width: '100%',
    maxWidth: 400,
    backgroundColor: Colors.bgCard,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: Layout.cardRadius,
    padding: Layout.cardPadding,
  },
  title: {
    fontSize: Typography.title,
    color: Colors.text,
    fontWeight: '700',
    marginBottom: Spacing.xs,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: Typography.body,
    color: Colors.textMuted,
    marginBottom: Spacing.xl,
    textAlign: 'center',
  },
  input: {
    backgroundColor: Colors.bgInput,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: Layout.cardRadius,
    padding: Spacing.md,
    color: Colors.text,
    fontSize: Typography.body,
    marginBottom: Spacing.md,
  },
  button: {
    backgroundColor: Colors.accent,
    paddingVertical: Spacing.md,
    borderRadius: Layout.cardRadius,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: '#fff',
    fontSize: Typography.body,
    fontWeight: '600',
  },
  errorText: {
    color: Colors.red,
    fontSize: Typography.caption,
    marginBottom: Spacing.md,
    textAlign: 'center',
  },
});
