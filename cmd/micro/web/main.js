// Minimal JS for reactive form submissions

document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('form[data-reactive]')?.forEach(function(form) {
        form.addEventListener('submit', async function(e) {
            e.preventDefault();
            const formData = new FormData(form);
            const params = {};
            for (const [key, value] of formData.entries()) {
                params[key] = value;
            }
            const action = form.getAttribute('action');
            const method = form.getAttribute('method') || 'POST';
            try {
                const resp = await fetch(action, {
                    method,
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(params)
                });
                const data = await resp.json();
                // Find or create a response container
                let respDiv = form.querySelector('.js-response');
                if (!respDiv) {
                    respDiv = document.createElement('div');
                    respDiv.className = 'js-response';
                    form.appendChild(respDiv);
                }
                respDiv.innerHTML = '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
            } catch (err) {
                alert('Error: ' + err);
            }
        });
    });
});
