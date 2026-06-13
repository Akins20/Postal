import { useState } from "react";
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from "react-native";

import { useMe } from "@/data/auth";
import {
  useAddMember,
  useMembers,
  ASSIGNABLE_ROLES,
  ROLE_LABELS,
  type AssignableRole,
  type Member,
} from "@/data/members";
import { ActivityFeed } from "@/features/members/activity-feed";
import { MemberChannelAccess } from "@/features/members/member-channel-access";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import type { NormalizedError } from "@/lib/api-error";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { FormField } from "@/ui/form-field";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

function roleTone(role: string): "accent" | "success" | "neutral" {
  if (role === "owner") return "accent";
  if (role === "admin") return "success";
  return "neutral";
}

function MemberRow({
  workspaceId,
  member,
  isYou,
  canManage,
}: {
  workspaceId: string;
  member: Member;
  isYou: boolean;
  canManage: boolean;
}) {
  const { palette } = usePalette();
  return (
    <View style={[styles.memberWrap, { borderBottomColor: palette.separator }]}>
      <View style={styles.row}>
        <View style={[styles.avatar, { backgroundColor: palette.accent }]}>
          <Text style={styles.avatarText}>{member.role.charAt(0).toUpperCase()}</Text>
        </View>
        <View style={{ flex: 1, minWidth: 0 }}>
          <Text style={[styles.name, { color: palette.fg }]} numberOfLines={1}>
            {member.user_id.slice(0, 8)}
            {isYou ? " (you)" : ""}
          </Text>
          <Text style={[styles.sub, { color: palette.fgSubtle }]}>
            {member.permissions?.length ?? 0} capabilities
          </Text>
        </View>
        <StatusPill tone={roleTone(member.role)}>
          {ROLE_LABELS[member.role] ?? member.role}
        </StatusPill>
      </View>
      {canManage && member.role !== "owner" && (
        <MemberChannelAccess workspaceId={workspaceId} userId={member.user_id} />
      )}
    </View>
  );
}

export default function MembersScreen() {
  const { palette } = usePalette();
  const { active } = useActiveWorkspace();
  const { data: me } = useMe();
  const { data: members, isPending, isError } = useMembers(active?.id);
  const add = useAddMember(active?.id ?? "");

  const [email, setEmail] = useState("");
  const [role, setRole] = useState<AssignableRole>("editor");
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);

  // Only owners/admins can manage members; gate the add form on the viewer's role.
  const myRole = members?.find((m) => m.user_id === me?.id)?.role;
  const canManage = myRole === "owner" || myRole === "admin";

  const submit = async () => {
    setError(null);
    setSent(false);
    try {
      await add.mutateAsync({ email: email.trim(), role });
      setEmail("");
      setSent(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <ScrollView style={{ backgroundColor: palette.surface }} contentContainerStyle={styles.content}>
      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Team</Text>
        {(!active || isPending) && <ActivityIndicator color={palette.fgSubtle} />}
        {isError && (
          <Text accessibilityRole="alert" style={[styles.sub, { color: palette.danger }]}>
            Couldn&apos;t load members. Pull to retry.
          </Text>
        )}
        {members?.map((m) => (
          <MemberRow
            key={m.user_id}
            workspaceId={active?.id ?? ""}
            member={m}
            isYou={m.user_id === me?.id}
            canManage={canManage}
          />
        ))}
      </Panel>

      {canManage && (
        <Panel>
          <Text style={[styles.cardTitle, { color: palette.fg }]}>Add a member</Text>
          <Text style={[styles.sub, { color: palette.fgMuted, marginBottom: space.sm }]}>
            They must already have a Postal account. They join with the role you pick.
          </Text>
          <FormField
            label="Email"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            inputMode="email"
            placeholder="teammate@example.com"
          />
          <Text style={[styles.roleLabel, { color: palette.fg }]}>Role</Text>
          <View style={styles.roleRow}>
            {ASSIGNABLE_ROLES.map((r) => {
              const selected = role === r;
              return (
                <Text
                  key={r}
                  onPress={() => setRole(r)}
                  style={[
                    styles.roleChip,
                    {
                      color: selected ? palette.accentFg : palette.fgMuted,
                      backgroundColor: selected ? palette.accent : "transparent",
                      borderColor: selected ? palette.accent : palette.separator,
                    },
                  ]}
                >
                  {ROLE_LABELS[r]}
                </Text>
              );
            })}
          </View>
          {sent && !error && (
            <Text style={[styles.sub, { color: palette.success }]}>Member added.</Text>
          )}
          {error && (
            <Text accessibilityRole="alert" style={[styles.sub, { color: palette.danger }]}>
              {error}
            </Text>
          )}
          <Button onPress={submit} loading={add.isPending} disabled={!email.trim()} style={styles.addBtn}>
            Add member
          </Button>
        </Panel>
      )}

      {canManage && active && <ActivityFeed workspaceId={active.id} />}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg },
  cardTitle: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  memberWrap: { borderBottomWidth: StyleSheet.hairlineWidth, paddingVertical: space.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.sm,
  },
  avatar: { width: 34, height: 34, borderRadius: 17, alignItems: "center", justifyContent: "center" },
  avatarText: { color: "#fff", fontSize: type.body, fontWeight: "700" },
  name: { fontSize: type.body, fontWeight: "600" },
  sub: { fontSize: type.caption },
  roleLabel: { fontSize: type.body, fontWeight: "600", marginTop: space.md, marginBottom: space.xs },
  roleRow: { flexDirection: "row", gap: space.sm },
  roleChip: {
    fontSize: type.caption,
    fontWeight: "600",
    borderWidth: StyleSheet.hairlineWidth,
    borderRadius: radius.md,
    paddingHorizontal: space.md,
    paddingVertical: space.sm,
    overflow: "hidden",
  },
  addBtn: { marginTop: space.md },
});
