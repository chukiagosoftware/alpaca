import {useEffect, useState} from 'react';
import {MapPin, Search, TrendingUp} from 'lucide-react';

// Update this to your API base URL
// If same domain, use empty string. If different domain, use full URL
const API_BASE_URL = '';

interface SearchResult {
    Hotel: string;
    City: string;
    Review: string;
    Rating: number;
    Distance: number;
    Address: string;
    map_url?: string;
    photo_thumb?: string;
    photo_full?: string;
}

interface SearchApiResponse {
    completion: SearchResult[];
    vector_count: number;
    safe_query: boolean;
    message?: string;
    model?: string;
    usage?: TokenUsage;
    timings?: {
        embedding_ms: number;
        vector_search_ms: number;
        safety_ms: number;
        metadata_ms: number;
        llm_completion_ms: number;
    };
}

interface TokenUsage {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens?: number;
}

export default function App() {
    const [searchQuery, setSearchQuery] = useState('quiet hotel with a good bar');
    const [selectedModel, setSelectedModel] = useState('');
    const [selectedRegion, setSelectedRegion] = useState('');
    const [selectedCity, setSelectedCity] = useState('');
    const [selectedRating, setSelectedRating] = useState('');
    const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
    const [locations, setLocations] = useState<Record<string, string[]>>({});
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [apiStats, setApiStats] = useState<SearchApiResponse['timings'] | null>(null);
    const [vectorCount, setVectorCount] = useState(0);
    const [safeQuery, setSafeQuery] = useState(true);
    const [modelUsed, setModelUsed] = useState('');
    const [usage, setUsage] = useState<TokenUsage | null>(null);
    const [userMessage, setUserMessage] = useState('');
    const [selectedPhoto, setSelectedPhoto] = useState<string>('');
    const [showForm, setShowForm] = useState(false);


    // Fetch locations on component mount
    useEffect(() => {
        fetchLocations();
    }, []);

    const fetchLocations = async () => {
        try {
            const url = `${API_BASE_URL}/api/locations`;
            const response = await fetch(url);

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();

            const locationMap: Record<string, string[]> = {};
            data.forEach((item: any) => {
                if (item.continent && Array.isArray(item.city_countries)) {
                    locationMap[item.continent] = item.city_countries;
                }
            });

            setLocations(locationMap);
        } catch (err) {
            const errorMsg = err instanceof Error ? err.message : 'Failed to load locations';
            setError(`Failed to load locations: ${errorMsg}`);
        }
    };

    const cities = selectedRegion ? locations[selectedRegion] || [] : [];

    // @ts-ignore
    const handleSearch = async (e?: React.FormEvent) => {
        if (e) e.preventDefault();

        setLoading(true);
        setError('');
        setUserMessage('');

        try {
            const params = new URLSearchParams();
            if (searchQuery) params.append('question', searchQuery);
            if (selectedRegion) params.append('continent', selectedRegion);
            if (selectedCity) params.append('citycountry', selectedCity);
            if (selectedRating) params.append('rating', selectedRating);
            if (selectedModel) params.append('llm', selectedModel);

            const url = `${API_BASE_URL}/api/search?${params.toString()}`;
            const response = await fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: params.toString(),
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data: SearchApiResponse = await response.json();

            setSearchResults(Array.isArray(data.completion) ? data.completion : []);
            setVectorCount(data.vector_count ?? 0);
            setSafeQuery(Boolean(data.safe_query));
            setApiStats(data.timings ?? null);
            setModelUsed(data.model ?? '');
            setUsage(data.usage ?? null);
            setUserMessage(data.message ?? '');
        } catch (err) {
            const errorMsg = err instanceof Error ? err.message : 'Unknown error';
            setError(`Search failed: ${errorMsg}`);
            setSearchResults([]);
            setVectorCount(0);
            setSafeQuery(true);
            setApiStats(null);
            setModelUsed('');
            setUsage(null);
            setUserMessage('');
        } finally {
            setLoading(false);
        }
    };

    const totalReviews = searchResults.length;
    const avgRating = searchResults.length > 0
        ? (searchResults.reduce((sum, r) => sum + Number(r.Rating), 0) / searchResults.length).toFixed(1)
        : '0.0';
    const uniqueCities = new Set(searchResults.map(r => r.City)).size;

    return (
        <div className="size-full flex flex-col bg-gray-50 p-4 md:p-6 gap-4 md:gap-6">
            <div className="flex flex-col md:flex-row gap-4 md:gap-6 flex-1 min-h-0">
                {/* Left Side - Search Form + Statistics */}
                {(showForm || window.innerWidth >= 768) && (  // Show on desktop or when toggled
                    <div className="w-full md:w-80 flex flex-col gap-4 md:gap-6 overflow-auto max-h-96 md:max-h-none">
                        <form onSubmit={handleSearch}
                              className="bg-white rounded-lg shadow-sm p-3 md:p-6 flex flex-col gap-3 md:gap-4">
                            <h2 className="flex items-center gap-2">
                                <Search className="w-5 h-5"/>
                                Hotel Search
                            </h2>

                            {error && (
                                <div
                                    className="px-3 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-600">
                                    {error}
                                </div>
                            )}

                            <div className="flex flex-col gap-2">
                                <label className="text-sm text-gray-600">Question</label>
                                <input
                                    type="text"
                                    placeholder="Search..."
                                    value={searchQuery}
                                    onChange={(e) => setSearchQuery(e.target.value)}
                                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                                />
                            </div>

                            <div className="flex flex-col gap-2">
                                <label className="text-sm text-gray-600">Region</label>
                                <select
                                    value={selectedRegion}
                                    onChange={(e) => {
                                        setSelectedRegion(e.target.value);
                                        setSelectedCity('');
                                    }}
                                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                                >
                                    <option value="">Select Region</option>
                                    {Object.keys(locations).map(region => (
                                        <option key={region} value={region}>{region}</option>
                                    ))}
                                </select>
                            </div>

                            <div className="flex flex-col gap-2">
                                <label className="text-sm text-gray-600">City, Country</label>
                                <select
                                    value={selectedCity}
                                    onChange={(e) => setSelectedCity(e.target.value)}
                                    disabled={!selectedRegion}
                                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                                >
                                    <option value="">Select City</option>
                                    {cities.map(city => (
                                        <option key={city} value={city}>{city}</option>
                                    ))}
                                </select>
                            </div>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="flex flex-col gap-2">
                                    <label className="text-sm text-gray-600">Rating</label>
                                    <select
                                        value={selectedRating}
                                        onChange={(e) => setSelectedRating(e.target.value)}
                                        className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                                        <option value="">ALL</option>
                                        <option value="5">5 Stars</option>
                                        <option value="4">4 Stars</option>
                                        <option value="3">3 Stars</option>
                                    </select>
                                </div>

                                <div className="flex flex-col gap-2">
                                    <label className="text-sm text-gray-600">AI Model</label>
                                    <select
                                        value={selectedModel}
                                        onChange={(e) => setSelectedModel(e.target.value)}
                                        className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                                    >
                                        <option value="">Auto</option>
                                        <option value="grok">Grok</option>
                                        <option value="gemini">Gemini</option>
                                        <option disabled value="openai">OpenAI</option>
                                    </select>
                                </div>
                            </div>

                            <button
                                type="submit"
                                disabled={loading}
                                className="mt-2 px-3 py-1.5 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:bg-gray-400 disabled:cursor-not-allowed"
                            >
                                {loading ? 'Searching...' : 'Search'}
                            </button>
                        </form>

                        {/* Statistics - 3 grouped sections */}
                        <div className="bg-white rounded-lg shadow-sm p-3 md:p-6">
                            <h2 className="mb-4 flex items-center gap-2">
                                <TrendingUp className="w-5 h-5"/>
                                Statistics
                            </h2>
                            {userMessage && (
                                <div
                                    className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded text-amber-700 text-sm">
                                    {userMessage}
                                </div>
                            )}

                            <div className="mb-6">
                                <p className="text-xs font-medium text-gray-500 mb-2">PERFORMANCE</p>
                                <table className="w-full text-sm">
                                    <tbody className="divide-y divide-gray-200">
                                    <tr>
                                        <td className="py-2 text-gray-600">Model</td>
                                        <td className="py-2 text-right font-medium">{modelUsed || '—'}</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">Query Embedding</td>
                                        <td className="py-2 text-right">{apiStats?.embedding_ms ?? 0} ms</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">LLM Prompt</td>
                                        <td className="py-2 text-right">{apiStats?.llm_completion_ms ?? 0} ms</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">Vector Search</td>
                                        <td className="py-2 text-right">{apiStats?.vector_search_ms ?? 0} ms</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">Vector Metadata</td>
                                        <td className="py-2 text-right">{apiStats?.metadata_ms ?? 0} ms</td>
                                    </tr>
                                    </tbody>
                                </table>
                            </div>

                            <div className="mb-6">
                                <p className="text-xs font-medium text-gray-500 mb-2">RESULTS</p>
                                <table className="w-full text-sm">
                                    <tbody className="divide-y divide-gray-200">
                                    <tr>
                                        <td className="py-2 text-gray-600">Average Rating</td>
                                        <td className="py-2 text-right">{avgRating} ⭐</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">Safe Query</td>
                                        <td className="py-2 text-right">{safeQuery ? 'Yes' : 'No'}</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">Total Results</td>
                                        <td className="py-2 text-right">{totalReviews}</td>
                                    </tr>
                                    <tr>
                                        <td className="py-2 text-gray-600">Unique Cities</td>
                                        <td className="py-2 text-right">{uniqueCities}</td>
                                    </tr>
                                    </tbody>
                                </table>
                            </div>

                            <div>
                                <p className="text-xs font-medium text-gray-500 mb-2">USAGE</p>
                                <table className="w-full text-sm">
                                    <tbody className="divide-y divide-gray-200">
                                    {usage && (
                                        <>
                                            <tr>
                                                <td className="py-2 text-gray-600">Prompt Tokens</td>
                                                <td className="py-2 text-right">{usage.prompt_tokens}</td>
                                            </tr>
                                            <tr>
                                                <td className="py-2 text-gray-600">Completion Tokens</td>
                                                <td className="py-2 text-right">{usage.completion_tokens}</td>
                                            </tr>
                                            {usage.total_tokens && (
                                                <tr>
                                                    <td className="py-2 text-gray-600">Total Tokens</td>
                                                    <td className="py-2 text-right">{usage.total_tokens}</td>
                                                </tr>
                                            )}
                                        </>
                                    )}
                                    <tr>
                                        <td className="py-2 text-gray-600">Vector Count</td>
                                        <td className="py-2 text-right">{vectorCount}</td>
                                    </tr>
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </div>
                )}

                {/* Right Panel - Search Results */}
                <div className="flex-1 flex flex-col min-h-0">
                    <div className="flex items-center justify-between mb-4">

                        <h2 className="text-xl font-semibold mb-4 flex items-center gap-2 text-gray-800">
                            <Search className="w-5 h-5"/>
                            Results
                        </h2>
                        <button
                            onClick={() => setShowForm(!showForm)}
                            className="md:hidden px-3 py-1 bg-gray-200 text-gray-700 rounded-md text-sm"
                        >
                            {showForm ? 'Hide Search' : 'Show Search'}
                        </button>
                    </div>

                    <div className="flex-1 overflow-auto space-y-4 md:space-y-6 pr-2">
                        {searchResults.length === 0 ? (
                            <div
                                className="h-full flex items-center justify-center text-gray-400 bg-white rounded-2xl border border-dashed border-gray-200">
                                <div className="text-center">
                                    <p className="text-lg mb-1">Find your next Hotel!</p>
                                    <p className="text-sm">Enter a query and click Search to get started</p>
                                </div>
                            </div>
                        ) : searchResults.map((result, index) => (
                            <div
                                key={index}
                                className="p-4 md:p-6 border border-gray-200 rounded-2xl hover:border-blue-300 transition-all bg-white">
                                {/* Hotel Name + Stars */}
                                <div className="flex justify-between items-start mb-5">
                                    <h3 className="text-xl font-semibold">{result.Hotel}</h3>
                                    <div className="text-3xl">{'⭐'.repeat(Math.round(result.Rating))}</div>
                                </div>

                                <div className="flex flex-col gap-6">
                                    {/* Address + Similarity */}
                                    <div>
                                        <div className="flex items-center gap-1 text-sm text-gray-600 mb-3">
                                            <MapPin className="w-4 h-4 flex-shrink-0"/>
                                            {result.Address} • {result.City}
                                        </div>
                                        <div
                                            className="inline-flex items-center gap-2 px-4 py-1.5 bg-emerald-50 text-emerald-700 rounded-full text-sm">
                                            <TrendingUp className="w-4 h-4"/>
                                            {(result.Distance * 100).toFixed(1)}%
                                        </div>
                                    </div>

                                    {/* Map + Photo */}
                                    <div className="flex gap-4">
                                        {result.map_url && (
                                            <div className="w-36 flex-shrink-0">
                                                <a
                                                    href={result.map_url}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="block h-28 border border-dashed border-gray-300 hover:border-red-300 rounded-2xl p-3 transition-colors group text-center"
                                                >
                                                    <MapPin className="w-7 h-7 text-red-500 mx-auto mb-2"/>
                                                    <p className="text-xs font-medium text-gray-700 group-hover:text-red-600">
                                                        View on map
                                                    </p>
                                                </a>
                                            </div>
                                        )}

                                        {result.photo_thumb && (
                                            <div className="flex-shrink-0">
                                                <img
                                                    src={result.photo_thumb}
                                                    alt="Hotel photo"
                                                    className="w-24 h-24 md:w-28 md:h-28 object-cover rounded-2xl shadow-md cursor-pointer hover:scale-105 transition-transform"
                                                    onClick={() => setSelectedPhoto(result.photo_full || result.photo_thumb!)}
                                                />
                                                <p className="text-center text-[10px] text-gray-400 mt-2">enlarge</p>
                                            </div>
                                        )}
                                    </div>
                                </div>

                                {/* Review Text */}
                                <p className="text-gray-700 leading-relaxed mt-6">{result.Review}</p>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            {/* Photo Modal */}
            {selectedPhoto && (
                <div
                    className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center p2 md:p-4"
                    onClick={() => setSelectedPhoto('')}
                >
                    <img
                        src={selectedPhoto}
                        alt="Enlarged photo"
                        className="max-h-[80vh] md:max-h-[90vh] max-w-full rounded-2xl shadow-2xl"
                        onClick={(e) => e.stopPropagation()}
                    />
                </div>
            )}
        </div>
    );
}