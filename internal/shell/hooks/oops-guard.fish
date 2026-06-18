# oops-guard — fish integration.
#
# Rebinds Enter so a destructive command is inspected before fish executes it.
# Declining returns you to the prompt with the command still on the line.
#
# Enable it by adding this to ~/.config/fish/config.fish:
#
#     oops-guard init fish | source
#
# Fails open: if oops-guard is missing or errors, your command runs normally.
# It blocks ONLY when you explicitly decline (exit code 10).

if not set -q _OOPS_GUARD_FISH_LOADED
    set -g _OOPS_GUARD_FISH_LOADED 1

    function _oops_guard_execute
        set -l cmd (commandline)
        if test -n "$cmd"; and command -q oops-guard
            command oops-guard guard --shell fish -- "$cmd"
            if test $status -eq 10
                commandline -f repaint
                return
            end
        end
        commandline -f execute
    end

    if status is-interactive
        bind \r _oops_guard_execute
        bind \n _oops_guard_execute
    end
end
