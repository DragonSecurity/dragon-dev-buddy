/**
 * Turn the pack's hooks off without editing the plugin.
 *
 *   DRAGON_BUDDY_HOOKS=off                  every hook in the pack does nothing
 *   DRAGON_BUDDY_DISABLED_HOOKS=a,b         just those, by the ids below
 *
 * Ids: session-start, project-memory, observe-gate, skill-tracker.
 *
 * A hook is edited out of a plugin by forking the plugin, and every release after
 * that is one the fork has to merge by hand -- so in practice a hook that is
 * wrong for someone gets the whole pack uninstalled instead. These are read from
 * the environment, which is where Claude Code's settings.json `env` block puts
 * them, so turning one off is a line of config that survives an upgrade.
 *
 * Lives under lib/ rather than beside the hooks because it is not one:
 * TestSessionStartHooksAreRegistered requires every top-level hooks/*.mjs to be
 * wired into hooks.json.
 */

const OFF = new Set(['off', '0', 'false', 'no', 'none']);

/** False when the user has switched this hook, or all of them, off. */
export function hookEnabled(id) {
  const all = String(process.env.DRAGON_BUDDY_HOOKS ?? '').trim().toLowerCase();
  if (OFF.has(all)) return false;
  const disabled = String(process.env.DRAGON_BUDDY_DISABLED_HOOKS ?? '')
    .split(',')
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean);
  return !disabled.includes(id);
}
