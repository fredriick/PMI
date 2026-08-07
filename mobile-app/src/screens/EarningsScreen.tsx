import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  ScrollView,
  RefreshControl,
  ActivityIndicator,
  StyleSheet,
} from 'react-native';
import { api } from '../services/api';
import { Colors, Spacing, Typography, Layout, width } from '../theme';

export default function EarningsScreen() {
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [amount, setAmount] = useState('0.00');
  const [period, setPeriod] = useState('Current Month');
  const [tier, setTier] = useState('Basic');
  const [sentGB, setSentGB] = useState(0);
  const [receivedGB, setReceivedGB] = useState(0);
  const [rateSent, setRateSent] = useState(0.5);
  const [rateReceived, setRateReceived] = useState(0.3);
  const [minPayout, setMinPayout] = useState(10.0);
  const [history, setHistory] = useState<Array<{ period: string; amount: number; tier: string }>>([]);

  const loadData = async () => {
    try {
      const [earnings] = await Promise.all([api.getEarnings()]);

      const p = earnings.payout || {};
      setAmount((p.amount || 0).toFixed(2));
      setPeriod(p.period || 'Current Month');
      setTier(p.tier || 'Basic');
      setSentGB(p.gb_sent || 0);
      setReceivedGB(p.gb_received || 0);

      if (p.tier && earnings.tiers?.length) {
        const tierInfo = earnings.tiers.find(t => t.name === p.tier);
        if (tierInfo) {
          setRateSent(tierInfo.rate_per_gb_sent);
          setRateReceived(tierInfo.rate_per_gb_recv);
        }
      } else {
        setRateSent(earnings.rates?.RatePerGBSent || 0.5);
        setRateReceived(earnings.rates?.RatePerGBReceived || 0.3);
      }
      setMinPayout(earnings.rates?.MinPayoutAmount || 10.0);
      setHistory(earnings.payout_history || []);
    } catch (err) {
      console.error('Earnings load error:', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 30000);
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

  const maxGB = Math.max(sentGB, receivedGB, 1);
  const sentPercent = (sentGB / maxGB) * 100;
  const receivedPercent = (receivedGB / maxGB) * 100;

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={Colors.accent} />
      }
    >
      <View style={styles.hero}>
        <Text style={styles.amount}>${amount}</Text>
        <Text style={styles.period}>{period}</Text>
        <View style={styles.tierBadge}>
          <Text style={styles.tierText}>{tier}</Text>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardHeader}>Payout Rates</Text>
        <View style={styles.row}>
          <Text style={styles.rowLabel}>Tier</Text>
          <Text style={styles.rowValue}>{tier}</Text>
        </View>
        <View style={styles.row}>
          <Text style={styles.rowLabel}>Per GB Sent</Text>
          <Text style={styles.rowValue}>${rateSent.toFixed(2)}</Text>
        </View>
        <View style={styles.row}>
          <Text style={styles.rowLabel}>Per GB Received</Text>
          <Text style={styles.rowValue}>${rateReceived.toFixed(2)}</Text>
        </View>
        <View style={styles.row}>
          <Text style={styles.rowLabel}>Minimum Payout</Text>
          <Text style={styles.rowValue}>${minPayout.toFixed(2)}</Text>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardHeader}>Bandwidth Breakdown</Text>
        <View style={styles.barContainer}>
          <Text style={styles.barLabel}>Sent</Text>
          <View style={styles.barTrack}>
            <View style={[styles.barFill, { width: `${sentPercent}%` }]} />
          </View>
          <Text style={styles.barValue}>{sentGB.toFixed(2)} GB</Text>
        </View>
        <View style={styles.barContainer}>
          <Text style={styles.barLabel}>Received</Text>
          <View style={styles.barTrack}>
            <View style={[styles.barFillGreen, { width: `${receivedPercent}%` }]} />
          </View>
          <Text style={styles.barValue}>{receivedGB.toFixed(2)} GB</Text>
        </View>
      </View>

      {history.length > 0 && (
        <View style={styles.card}>
          <Text style={styles.cardHeader}>History</Text>
          {history.map((item, index) => (
            <View key={index} style={styles.historyItem}>
              <Text style={styles.historyPeriod}>{item.period}</Text>
              <Text style={styles.historyAmount}>
                ${item.amount.toFixed(2)} · {item.tier}
              </Text>
            </View>
          ))}
        </View>
      )}
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
  hero: {
    backgroundColor: Colors.bgCard,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: Layout.cardRadius,
    padding: Spacing.xl,
    alignItems: 'center',
    marginBottom: Spacing.lg,
  },
  amount: {
    fontSize: 44,
    color: Colors.green,
    fontWeight: '700',
    fontFamily: 'monospace',
  },
  period: {
    fontSize: Typography.body,
    color: Colors.textMuted,
    marginTop: Spacing.xs,
    fontWeight: '500',
  },
  tierBadge: {
    backgroundColor: Colors.accent,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.xs,
    borderRadius: 20,
    marginTop: Spacing.md,
  },
  tierText: {
    color: '#fff',
    fontSize: Typography.small,
    fontWeight: '600',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
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
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: Spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(59, 130, 246, 0.06)',
  },
  rowLabel: {
    fontSize: Typography.body,
    color: Colors.textMuted,
  },
  rowValue: {
    fontSize: Typography.body,
    color: Colors.text,
    fontWeight: '500',
    fontFamily: 'monospace',
  },
  barContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm,
    marginBottom: Spacing.md,
  },
  barLabel: {
    width: 70,
    fontSize: Typography.small,
    color: Colors.textMuted,
  },
  barTrack: {
    flex: 1,
    height: 8,
    backgroundColor: Colors.bgInput,
    borderRadius: 4,
    borderWidth: 1,
    borderColor: Colors.border,
    overflow: 'hidden',
  },
  barFill: {
    height: '100%',
    backgroundColor: Colors.accent,
    borderRadius: 4,
  },
  barFillGreen: {
    height: '100%',
    backgroundColor: Colors.green,
    borderRadius: 4,
  },
  barValue: {
    width: 80,
    textAlign: 'right',
    fontSize: Typography.caption,
    color: Colors.text,
    fontWeight: '500',
    fontFamily: 'monospace',
  },
  historyItem: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    padding: Spacing.sm,
    backgroundColor: Colors.bgInput,
    borderRadius: 8,
    marginBottom: Spacing.sm,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  historyPeriod: {
    fontSize: Typography.caption,
    color: Colors.text,
  },
  historyAmount: {
    fontSize: Typography.caption,
    color: Colors.textMuted,
  },
});
