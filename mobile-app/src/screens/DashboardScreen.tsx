import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  RefreshControl,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { api } from '../services/api';
import { Colors, Spacing, Typography, Layout, width } from '../theme';

interface Metrics {
  status: string;
  battery: string;
  cpu: string;
  load: string;
  country: string;
  isp: string;
  health: string;
}

export default function DashboardScreen() {
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [metrics, setMetrics] = useState<Metrics>({
    status: '--',
    battery: '--',
    cpu: '--',
    load: '--',
    country: '--',
    isp: '--',
    health: '--',
  });
  const [online, setOnline] = useState(false);

  const loadData = async () => {
    try {
      const [status, health] = await Promise.all([
        api.getStatus(),
        api.getHealth().catch(() => null),
      ]);

      const node = status.node;
      setOnline(node.online !== false);

      setMetrics({
        status: node.online !== false ? 'Online' : 'Offline',
        battery: node.battery ? `${node.battery}%` : '--',
        cpu: node.cpu_usage ? `${node.cpu_usage.toFixed(1)}%` : '--',
        load: String(status.load || 0),
        country: node.country || '--',
        isp: node.isp || '--',
        health: health?.overall_score ? health.overall_score.toFixed(0) : '--',
      });
    } catch (err) {
      console.error('Dashboard load error:', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 15000);
    return () => clearInterval(interval);
  }, []);

  const onRefresh = () => {
    setRefreshing(true);
    loadData();
  };

  if (loading) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color={Colors.accent} />
      </View>
    );
  }

  const MetricCard = ({ label, value }: { label: string; value: string }) => (
    <View style={styles.metricCard}>
      <Text style={styles.metricLabel}>{label}</Text>
      <Text style={styles.metricValue}>{value}</Text>
    </View>
  );

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={Colors.accent} />
      }
    >
      <View style={styles.header}>
        <View>
          <Text style={styles.headerTitle}>Dashboard</Text>
          <Text style={styles.headerSubtitle}>Node Status</Text>
        </View>
        <View style={styles.statusBadge}>
          <View style={[styles.statusDot, online ? styles.statusOnline : styles.statusOffline]} />
          <Text style={[styles.statusText, online ? styles.statusOnlineText : styles.statusOfflineText]}>
            {online ? 'Online' : 'Offline'}
          </Text>
        </View>
      </View>

      <View style={styles.metricGrid}>
        <MetricCard label="Status" value={metrics.status} />
        <MetricCard label="Battery" value={metrics.battery} />
        <MetricCard label="CPU" value={metrics.cpu} />
        <MetricCard label="Load" value={metrics.load} />
        <MetricCard label="Country" value={metrics.country} />
        <MetricCard label="ISP" value={metrics.isp} />
        <View style={styles.metricCard}>
          <Text style={styles.metricLabel}>Health</Text>
          <Text style={[styles.metricValue, { color: Colors.green }]}>
            {metrics.health}
          </Text>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardHeader}>Quick Actions</Text>
        <View style={styles.actionRow}>
          <TouchableOpacity
            style={styles.actionButton}
            onPress={() =>
              Alert.alert('Disconnect', 'Use Settings tab to disconnect and clear session.')
            }
          >
            <Text style={styles.actionButtonText}>Disconnect</Text>
          </TouchableOpacity>
        </View>
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
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: Spacing.lg,
  },
  headerTitle: {
    fontSize: Typography.title,
    color: Colors.text,
    fontWeight: '700',
  },
  headerSubtitle: {
    fontSize: Typography.caption,
    color: Colors.textMuted,
    marginTop: 2,
  },
  statusBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.bgCard,
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
  statusOnline: {
    backgroundColor: Colors.green,
  },
  statusOffline: {
    backgroundColor: Colors.red,
  },
  statusText: {
    fontSize: Typography.caption,
    fontWeight: '600',
  },
  statusOnlineText: {
    color: Colors.green,
  },
  statusOfflineText: {
    color: Colors.red,
  },
  metricGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.sm,
    marginBottom: Spacing.lg,
  },
  metricCard: {
    flex: 1,
    minWidth: (width - Spacing.lg * 3) / 2,
    backgroundColor: Colors.bgCard,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: Layout.cardRadius,
    padding: Spacing.md,
    marginBottom: Spacing.sm,
  },
  metricLabel: {
    fontSize: Typography.small,
    color: Colors.textMuted,
    textTransform: 'uppercase',
    letterSpacing: 0.6,
    marginBottom: Spacing.xs,
    fontWeight: '500',
  },
  metricValue: {
    fontSize: 20,
    color: Colors.accentBright,
    fontWeight: '600',
    fontFamily: 'monospace',
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
  actionRow: {
    flexDirection: 'row',
    gap: Spacing.sm,
  },
  actionButton: {
    flex: 1,
    backgroundColor: Colors.accent,
    paddingVertical: Spacing.md,
    borderRadius: Layout.cardRadius,
    alignItems: 'center',
  },
  actionButtonText: {
    color: '#fff',
    fontSize: Typography.body,
    fontWeight: '600',
  },
});
