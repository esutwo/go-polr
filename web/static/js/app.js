// go-polr JavaScript

/**
 * Copy short URL to clipboard
 */
function copyToClipboard() {
    const input = document.getElementById('short-url');
    if (!input) return;

    input.select();
    input.setSelectionRange(0, 99999); // For mobile devices

    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(input.value).then(() => {
            showCopyFeedback();
        }).catch(err => {
            console.error('Failed to copy:', err);
            fallbackCopy(input);
        });
    } else {
        fallbackCopy(input);
    }
}

/**
 * Fallback copy method for older browsers
 */
function fallbackCopy(input) {
    try {
        document.execCommand('copy');
        showCopyFeedback();
    } catch (err) {
        console.error('Fallback copy failed:', err);
        alert('Press Ctrl+C to copy');
    }
}

/**
 * Show copy feedback to user
 */
function showCopyFeedback() {
    const btn = document.querySelector('.result-url button');
    if (btn) {
        const originalText = btn.textContent;
        btn.textContent = 'Copied!';
        btn.disabled = true;
        setTimeout(() => {
            btn.textContent = originalText;
            btn.disabled = false;
        }, 2000);
    }
}

/**
 * Initialize on page load
 */
document.addEventListener('DOMContentLoaded', function() {
    // Auto-focus on main input
    const urlInput = document.getElementById('url');
    if (urlInput && !urlInput.value) {
        urlInput.focus();
    }

    // Auto-select short URL on click
    const shortUrlInput = document.getElementById('short-url');
    if (shortUrlInput) {
        shortUrlInput.addEventListener('click', function() {
            this.select();
        });
    }

    // Confirm dialogs for destructive actions
    const deleteButtons = document.querySelectorAll('[data-confirm]');
    deleteButtons.forEach(btn => {
        btn.addEventListener('click', function(e) {
            if (!confirm(this.dataset.confirm)) {
                e.preventDefault();
            }
        });
    });
});
