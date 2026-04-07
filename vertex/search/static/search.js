document.getElementById('searchForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const form = e.target;
    const submitButton = form.querySelector('button');
    const resultsEl = document.getElementById('results');
    const metricsEl = document.getElementById('metrics');

    if (!resultsEl || !metricsEl) return;

    if (form.dataset.submitting === 'true') return;
    form.dataset.submitting = 'true';
    submitButton.disabled = true;
    submitButton.textContent = 'Searching...';

    const formData = new FormData(form);

    try {
        showLoading(true);
        metricsEl.innerHTML = 'Searching...';

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 60000);

        const response = await fetch('/api/search', {
            method: 'POST',
            body: formData,
            signal: controller.signal
        });

        clearTimeout(timeoutId);

        if (!response.ok) throw new Error(`Server error ${response.status}`);

        const data = await response.json();

        // Display review cards in main area
        resultsEl.innerHTML = '';


        data.completion.forEach(review => {
            const header = `${review.Hotel || 'Unknown Hotel'} — ${review.City || ''}`;
            const body = review.Review || 'No review text';
            const footer = [];

            if (review.Rating !== undefined) footer.push(`⭐ ${review.Rating}`);
            if (review.Distance !== undefined) footer.push(`Distance: ${review.Distance.toFixed(3)}`);
            if (review.Address) footer.push(review.Address);

            resultsEl.innerHTML += `
                <div class="review">
                    <div class="review-header">${header}</div>
                    <div class="review-body">${body}</div>
                    <div class="review-footer">${footer.join(' | ')}</div>
                </div>`;
        });


        // Display metrics in the bottom panel
        if (data.timings) {
            let html = '<strong>Performance:</strong><br>';
            Object.entries(data.timings).forEach(([key, value]) => {
                const label = key.replace('_ms', '').replace('_', ' ');
                html += `${label}: ${value}ms<br>`;
            });
            metricsEl.innerHTML = html;
        } else {
            metricsEl.innerHTML = 'Metrics not available';
        }

    } catch (err) {
        console.error('Search error:', err);
        const msg = err.name === 'AbortError'
            ? 'Request timed out.'
            : 'Search failed. Please try again.';
        metricsEl.innerHTML = `<span style="color:red">${msg}</span>`;
        resultsEl.innerHTML = `<p>${msg}</p>`;
    } finally {
        form.dataset.submitting = 'false';
        submitButton.disabled = false;
        submitButton.textContent = 'Search';
        showLoading(false);
    }
});

function showLoading(show) {
    let spinner = document.getElementById('loadingSpinner');
    if (!spinner) {
        spinner = document.createElement('div');
        spinner.id = 'loadingSpinner';
        spinner.innerHTML = '⟳';
        spinner.style.cssText = 'text-align:center; padding:40px; font-size:2.5rem; color:#4299e1;';
    }
    const resultsArea = document.getElementById('results');
    if (show) {
        resultsArea.parentNode.prepend(spinner);
        spinner.style.display = 'block';
    } else if (spinner.parentNode) {
        spinner.style.display = 'none';
    }
}