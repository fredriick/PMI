import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  ScrollView,
  Switch,
  TouchableOpacity,
  Alert,
  StyleSheet,
} from 'react-native';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { Colors, Spacing, Typography, Layout } from '../theme';

export default function SettingsScreen() {
  const { logout } = useAuth();
  const [nodeId, setNodeId] = useState('--');
  const [consent, setConsent] = useState(true);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const status = await api.getStatus();
      setNodeId(status.node.id || '--');
    } catch (err) {
      console.error('Settings load error:', err);
    }
  };

  const handleConsentToggle = async (value: boolean) => {
    setConsent(value);
    try {
      await api.setConsent(value);
    } catch (err: any) {
      setConsent(!value);
      Alert.alert('Error', err.message || 'Failed to update consent');
    }
  };

  const handleDisconnect = () => {
    Alert.alert('Disconnect', 'Are you sure you want to disconnect and clear your session?', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Disconnect', style: 'destructive', onPress: logout },
    ]);
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.card}>
        <Text style={styles.cardHeader}>Participation</Text>
        <View style={styles.row}>
          <View>
            <Text style={styles.settingLabel}>Node Active</Text>
            <Text style={styles.settingDesc}>Enable or disable your node on the network</Text>
          </View>
          <Switch
            value={consent}
            onValueChange={handleConsentToggle}
            trackColor={{ false: Colors.bgInput, true: Colors.accent }}
            thumbColor="#fff"
          />
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardHeader}>Session</Text>
        <View style={styles.row}>
          <View>
            <Text style={styles.settingLabel}>Node ID</Text>
            <Text style={styles.settingDescMono}>{nodeId}</Text>
          </View>
        </View>
        <TouchableOpacity style={styles.dangerButton} onPress={handleDisconnect}>
          <Text style={styles.dangerButtonText}>Disconnect & Clear Session</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardHeader}>About</Text>
        <Text style={styles.aboutText}>ProxyMesh Peer v1.0</Text>
        <Text style={styles.aboutDesc}>Manage your residential node and track earnings.</Text>
        <Text style={styles.aboutSmall}>Built with React Native and Expo.</Text>
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
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: Spacing.sm,
  },
  settingLabel: {
    fontSize: Typography.body,
    color: Colors.text,
    fontWeight: '500',
  },
  settingDesc: {
    fontSize: Typography.small,
    color: Colors.textMuted,
    marginTop: 2,
  },
  settingDescMono: {
    fontSize: Typography.body,
    color: Colors.accentBright,
    fontFamily: 'monospace',
    marginTop: 2,
  },
  dangerButton: {
    backgroundColor: Colors.red,
    paddingVertical: Spacing.md,
    borderRadius: Layout.cardRadius,
    alignItems: 'center',
    marginTop: Spacing.md,
  },
  dangerButtonText: {
    color: '#fff',
    fontSize: Typography.body,
    fontWeight: '600',
  },
  aboutText: {
    fontSize: Typography.body,
    color: Colors.accentBright,
    fontWeight: '600',
    marginBottom: Spacing.xs,
  },
  aboutDesc: {
    fontSize: Typography.body,
    color: Colors.textMuted,
    marginBottom: Spacing.sm,
  },
  aboutSmall: {
    fontSize: Typography.small,
    color: Colors.textMuted,
  },
});
