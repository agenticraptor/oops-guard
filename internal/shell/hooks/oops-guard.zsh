# oops-guard — zsh integration.
#
# Wraps the Enter key (the accept-line widget) so a destructive command is
# inspected the instant you press Return, before zsh runs it. Declining leaves
# the command in your edit buffer so you can fix or clear it.
#
# Enable it by adding this to ~/.zshrc:
#
#     eval "$(oops-guard init zsh)"
#
# Fails open: if oops-guard is missing or errors, your command runs normally.
# It blocks ONLY when you explicitly decline (exit code 10).

if [[ -n ${_OOPS_GUARD_ZSH_LOADED:-} ]]; then
  return 0 2>/dev/null
fi
typeset -g _OOPS_GUARD_ZSH_LOADED=1

_oops-guard-accept-line() {
  # Only consult oops-guard when it's installed; otherwise behave like Enter.
  if [[ -n $BUFFER ]] && (( $+commands[oops-guard] )); then
    command oops-guard guard --shell zsh -- "$BUFFER"
    if (( $? == 10 )); then
      # Declined: keep the buffer, repaint the prompt, do not execute.
      zle reset-prompt
      return 0
    fi
  fi
  zle .accept-line
}

zle -N accept-line _oops-guard-accept-line
