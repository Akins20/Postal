import { ActivityIndicator, Linking, ScrollView, StyleSheet, Text, View } from "react-native";

import { useLedger, useWallet, type LedgerEntry } from "@/data/billing";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";

// Top-ups happen on the web (Play billing policy keeps the app read-only).
const WEB_WALLET_URL = "https://postal.lettstv.com/wallet";

const KIND_LABEL: Record<string, string> = {
  topup: "Top-up",
  publish_charge: "X publish",
  refund: "Refund",
  adjustment: "Adjustment",
};

function Row({ entry }: { entry: LedgerEntry }) {
  const { palette } = usePalette();
  const positive = entry.credits > 0;
  return (
    <View style={[styles.ledgerRow, { borderBottomColor: palette.separator }]}>
      <View style={{ flex: 1 }}>
        <Text style={[styles.kind, { color: palette.fg }]}>{KIND_LABEL[entry.kind] ?? entry.kind}</Text>
        <Text style={[styles.sub, { color: palette.fgSubtle }]}>
          {new Date(entry.created_at).toLocaleString()}
        </Text>
      </View>
      <Text style={[styles.amount, { color: positive ? palette.success : palette.fg }]}>
        {positive ? "+" : ""}
        {entry.credits}
      </Text>
    </View>
  );
}

export default function WalletScreen() {
  const { palette } = usePalette();
  const { active } = useActiveWorkspace();
  const { data: wallet, isPending } = useWallet(active?.id);
  const { data: ledger } = useLedger(active?.id);

  const costs = wallet?.publish_costs ?? {};
  const tiers = [
    { label: "Plain X post", value: costs.twitter },
    { label: "With media", value: costs.twitter_media },
    { label: "With a link", value: costs.twitter_url },
  ].filter((t) => Boolean(t.value));

  return (
    <ScrollView style={{ backgroundColor: palette.surface }} contentContainerStyle={styles.content}>
      {(!active || isPending) && <ActivityIndicator color={palette.fgSubtle} />}
      {wallet && (
        <>
          <Panel>
            <Text style={[styles.sub, { color: palette.fgMuted }]}>Balance</Text>
            <Text style={[styles.balance, { color: palette.fg }]}>
              {wallet.balance}
              <Text style={[styles.sub, { color: palette.fgSubtle }]}> credits</Text>
            </Text>
            <Text style={[styles.sub, { color: palette.fgSubtle }]}>
              ${(wallet.balance / 100).toFixed(2)} of publishing power
            </Text>
          </Panel>

          {tiers.length > 0 && (
            <Panel>
              <Text style={[styles.cardTitle, { color: palette.fg }]}>Credits per X post</Text>
              {tiers.map((t) => (
                <View key={t.label} style={styles.tierRow}>
                  <Text style={[styles.sub, { color: palette.fgMuted }]}>{t.label}</Text>
                  <Text style={[styles.tierVal, { color: palette.fg }]}>{t.value}</Text>
                </View>
              ))}
            </Panel>
          )}

          <View style={[styles.note, { backgroundColor: `${palette.accent}12`, borderColor: `${palette.accent}33` }]}>
            <Text style={[styles.sub, { color: palette.fg }]}>
              Top-ups are handled on the Postal web app. Only publishing to X uses credits; every
              other platform is free.
            </Text>
            <Button onPress={() => Linking.openURL(WEB_WALLET_URL)} style={styles.topUpBtn}>
              Top up on the web
            </Button>
          </View>

          <Panel>
            <Text style={[styles.cardTitle, { color: palette.fg }]}>History</Text>
            {(!ledger || ledger.length === 0) && (
              <Text style={[styles.sub, { color: palette.fgMuted }]}>No movements yet.</Text>
            )}
            {ledger?.map((e) => <Row key={e.id} entry={e} />)}
          </Panel>
        </>
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg },
  sub: { fontSize: type.caption + 1 },
  cardTitle: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  balance: { fontSize: 40, fontWeight: "700", letterSpacing: -1, fontVariant: ["tabular-nums"], marginVertical: 2 },
  tierRow: { flexDirection: "row", justifyContent: "space-between", paddingVertical: space.xs },
  tierVal: { fontSize: type.body, fontWeight: "600", fontVariant: ["tabular-nums"] },
  note: { padding: space.md, borderRadius: radius.md, borderWidth: 1, gap: space.sm },
  topUpBtn: { marginTop: space.xs },
  ledgerRow: { flexDirection: "row", alignItems: "center", paddingVertical: space.md, borderBottomWidth: StyleSheet.hairlineWidth },
  kind: { fontSize: type.body, fontWeight: "500" },
  amount: { fontSize: type.body, fontWeight: "700", fontVariant: ["tabular-nums"] },
});
