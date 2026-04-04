document.getElementById('searchForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const form = e.target;
    const submitButton = form.querySelector('button');
    const resultsEl = document.getElementById('results');
    const metricsEl = document.getElementById('metrics');
    const mapEl = document.getElementById('map');

    if (!resultsEl || !metricsEl) return;

    // Prevent multiple submissions
    if (form.dataset.submitting === 'true') return;
    form.dataset.submitting = 'true';
    submitButton.disabled = true;
    submitButton.textContent = 'Searching...';

    const formData = new FormData(form);

    try {
        showLoading(true);
        updateMetrics("Searching vector database...");

        // Load map early
        const cityCountry = document.getElementById('citycountry')?.value || '';
        if (mapEl) loadCityMap(cityCountry);

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 45000); // 45 second timeout

        const response = await fetch('/api/search', {
            method: 'POST',
            body: formData,
            signal: controller.signal
        });

        clearTimeout(timeoutId);

        if (!response.ok) throw new Error(`Server error ${response.status}`);

        const data = await response.json();

        const vectorCount = data.vector_count || data.results?.length || 0;
        updateMetrics(`${vectorCount} vector search results. Please hold for the best hotels`);

        resultsEl.innerHTML = data.completion || '<p>No results found.</p>';

    } catch (err) {
        console.error('Search error:', err);
        const msg = err.name === 'AbortError'
            ? 'Request timed out. Please try again.'
            : 'Search failed. Please check your connection and try again.';
        updateMetrics(msg);
        resultsEl.innerHTML = `<p>${msg}</p>`;
    } finally {
        form.dataset.submitting = 'false';
        submitButton.disabled = false;
        submitButton.textContent = 'Search';
        showLoading(false);
    }
});

// Helper functions
function showLoading(show) {
    let spinner = document.getElementById('loadingSpinner');
    if (!spinner) {
        spinner = document.createElement('div');
        spinner.id = 'loadingSpinner';
        spinner.style.cssText = 'text-align:center; padding:30px; font-size:1.8rem;';
        spinner.innerHTML = '⟳';
    }
    if (show) {
        document.getElementById('results').parentNode.prepend(spinner);
        spinner.style.display = 'block';
    } else {
        spinner.style.display = 'none';
    }
}

function updateMetrics(html) {
    const el = document.getElementById('metrics');
    if (el) el.innerHTML = `<p>${html}</p>`;
}
