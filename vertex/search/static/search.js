document.getElementById('searchForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    const response = await fetch('/search', { method: 'POST', body: formData });
    const data = await response.json();

    // Debug: Log the received data
    console.log('Data received:', data);

    // Update results in right column
    let resultsHtml = '';
    resultsHtml += `<p><strong>Completion:</strong> ${data.completion}</p>`;
    document.getElementById('results').innerHTML = resultsHtml;

    // Update metrics in bottom panel
    document.getElementById('metrics').innerHTML = `
        <p><strong>Embedding Time:</strong> ${data.timings.embedding_ms} ms</p>
        <p><strong>Vector Search Time:</strong> ${data.timings.vector_search_ms} ms</p>
        <p><strong>LLM Completion Time:</strong> ${data.timings.llm_completion_ms} ms</p>
    `;
});