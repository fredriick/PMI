import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  ScrollView,
  TextInput,
  TouchableOpacity,
  Alert,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { api } from '../services/api';
import { Colors, Spacing, Typography, Layout } from '../theme';

interface NodeInfo {
  id: string;
  country?: string;
  city?: string;
  os?: string;
  node_type?: string;
  ip?: string;
  last_seen?: string;
  reputation?: number;
  online?: boolean;
}

export default function NodeManagementScreen() {
  const [nodeInfo, setNodeInfo] = useState<NodeInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [country, setCountry] = useState('');
  const [city, setCity] = useState('');
  const [os, setOs] = useState('');

  useEffect(() => {
    loadNodeInfo();
  }, []);

  const loadNodeInfo = async () => {
    try {
      const status = await api.getStatus();
      const node = status.node;
      setNodeInfo({
        id: node.id || '--',
        country: node.country || '',
        city: node.city || '',
        os: node.os || '',
        node_type: node.node_type || 'residential',
        ip: node.ip || '--',
        last_seen: node.last_seen || '--',
        reputation: node.reputation || 0,
        online: node.online !== false,
      });
      setCountry(node.country || '');
      setCity(node.city || '');
      setOs(node.os || '');
    } catch (err) {
      console.error('Failed to load node info:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleUpdate = async () => {
    setSaving(true);
    try {
      await api.updateNodeDetails({
        country: country || undefined,
        city: city || undefined,
        os: os || undefined,
      });
      Alert.alert('Success', 'Node details updated');
      await loadNodeInfo();
    } catch (err: any) {
      Alert.alert('Error', err.message || 'Failed to update node');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color={Colors.accent} />
      </View>
    );
  }

  if (!nodeInfo) {
    return (
      <View style={styles.centerContainer}>
        <Text style={styles.errorText}>Failed to load node information</Text>
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.card}>
        <Text style={styles.cardHeader}>Node Information</Text>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Node ID</Text>
          <Text style={styles.infoValue}>{nodeInfo.id}</Text>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Type</Text>
          <Text style={styles.infoValue}>{nodeInfo.node_type}</Text>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>IP Address</Text>
          <Text style={styles.infoValue}>{nodeInfo.ip}</Text>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Status</Text>
          <View style={[styles.statusBadge, nodeInfo.online ? styles.statusOnline : styles.statusOffline]}>
            <View style={[styles.statusDot, nodeInfo.online ? styles.dotOnline : styles.dotOffline]} />
            <Text style={[styles.statusText, nodeInfo.online ? styles.textOnline : styles.textOffline]}>
              {nodeInfo.online ? 'Online' : 'Offline'}
            </Text>
          </View>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Reputation</Text>
          <Text style={styles.infoValue}>{(nodeInfo.reputation || 0).toFixed(0)}/100</Text>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardHeader}>Update Details</Text>
        <Text style={styles.inputLabel}>Country</Text>
        <TextInput
          style={styles.input}
          value={country}
          onChangeText={setCountry}
          placeholder="US"
          placeholderTextColor={Colors.textMuted}
          autoCapitalize="characters"
          maxLength={2}
        />
        <Text style={styles.inputLabel}>City</Text>
        <TextInput
          style={styles.input}
          value={city}
          onChangeText={setCity}
          placeholder="New York"
          placeholderTextColor={Colors.textMuted}
        />
        <Text style={styles.inputLabel}>OS</Text>
        <TextInput
          style={styles.input}
          value={os}
          onChangeText={setOs}
          placeholder="Ubuntu 22.04"
          placeholderTextColor={Colors.textMuted}
        />
        <TouchableOpacity
          style={[styles.button, saving && styles.buttonDisabled]}
          onPress={handleUpdate}
          disabled={saving}
        >
          {saving ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Update Node</Text>
          )}
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.bg,
  },
  content: {
    padding: Layout.screenPadding,
    paddingBottom: 100,
  },
  centerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: Colors.bg,
  },
  card: {
    backgroundColor: Colors.bgCard,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: Layout.cardRadius,
    padding: Layout.cardPadding,
    marginBottom: Spacing.md,
  },
  cardHeader: {
    fontSize: Typography.caption,
    color: Colors.textMuted,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    marginBottom: Spacing.md,
    fontWeight: '500',
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: Spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(59, 130, 246, 0.06)',
  },
  infoLabel: {
    fontSize: Typography.body,
    color: Colors.textMuted,
  },
  infoValue: {
    fontSize: Typography.body,
    color: Colors.text,
    fontWeight: '500',
    fontFamily: 'monospace',
  },
  statusBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.bgInput,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: 20,
    paddingVertical: 4,
    paddingHorizontal: 12,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: Spacing.sm,
  },
  dotOnline: {
    backgroundColor: Colors.green,
  },
  dotOffline: {
    backgroundColor: Colors.red,
  },
  statusText: {
    fontSize: Typography.caption,
    fontWeight: '600',
  },
  textOnline: {
    color: Colors.green,
  },
  textOffline: {
    color: Colors.red,
  },
  inputLabel: {
    fontSize: Typography.body,
    color: Colors.textMuted,
    marginBottom: Spacing.xs,
    marginTop: Spacing.sm,
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
    marginTop: Spacing.md,
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
    fontSize: Typography.body,
    textAlign: 'center',
  },
});
