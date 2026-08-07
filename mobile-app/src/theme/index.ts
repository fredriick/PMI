import { StyleSheet, Dimensions } from 'react-native';

const { width } = Dimensions.get('window');

export const Colors = {
  bg: '#050810',
  bgCard: 'rgba(10, 14, 26, 0.92)',
  bgInput: 'rgba(10, 14, 26, 0.8)',
  border: 'rgba(59, 130, 246, 0.15)',
  text: '#e8eaed',
  textMuted: '#6b7280',
  accent: '#3b82f6',
  accentBright: '#60a5fa',
  accentDim: 'rgba(59, 130, 246, 0.12)',
  accentGlow: 'rgba(59, 130, 246, 0.5)',
  cyan: '#06b6d4',
  green: '#10b981',
  red: '#ef4444',
  orange: '#f59e0b',
};

export const Spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
};

export const Typography = {
  title: 24,
  heading: 18,
  body: 14,
  caption: 12,
  small: 11,
};

export const Layout = {
  cardPadding: Spacing.lg,
  cardRadius: 12,
  screenPadding: Spacing.lg,
};

export default StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.bg,
  },
  screen: {
    flex: 1,
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
  title: {
    fontSize: Typography.heading,
    color: Colors.text,
    fontWeight: '600',
  },
  text: {
    fontSize: Typography.body,
    color: Colors.text,
  },
  muted: {
    fontSize: Typography.caption,
    color: Colors.textMuted,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: Spacing.sm,
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
  metricGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.sm,
  },
  metricCard: {
    flex: 1,
    minWidth: (width - Spacing.lg * 3) / 2,
    backgroundColor: Colors.bgCard,
    borderColor: Colors.border,
    borderWidth: 1,
    borderRadius: Layout.cardRadius,
    padding: Spacing.md,
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
  button: {
    backgroundColor: Colors.accent,
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.lg,
    borderRadius: Layout.cardRadius,
    alignItems: 'center',
    justifyContent: 'center',
  },
  buttonText: {
    color: '#fff',
    fontSize: Typography.body,
    fontWeight: '600',
  },
  dangerButton: {
    backgroundColor: Colors.red,
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
  statusDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    marginRight: Spacing.sm,
  },
  statusOnline: {
    backgroundColor: Colors.green,
    shadowColor: Colors.green,
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.8,
    shadowRadius: 6,
  },
  statusOffline: {
    backgroundColor: Colors.red,
    shadowColor: Colors.red,
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.8,
    shadowRadius: 6,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: Spacing.lg,
    borderBottomWidth: 1,
    borderBottomColor: Colors.border,
  },
  nav: {
    flexDirection: 'row',
    backgroundColor: Colors.bgCard,
    borderTopWidth: 1,
    borderTopColor: Colors.border,
    paddingBottom: 8,
  },
  navItem: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: 8,
  },
  navLabel: {
    fontSize: 10,
    color: Colors.textMuted,
    marginTop: 2,
  },
  tabContent: {
    flex: 1,
    padding: Layout.screenPadding,
  },
  skeleton: {
    height: 18,
    backgroundColor: 'rgba(59,130,246,0.12)',
    borderRadius: 6,
    marginBottom: 8,
  },
  errorText: {
    color: Colors.red,
    textAlign: 'center',
    padding: 32,
  },
});

export { width };
