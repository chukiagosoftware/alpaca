import {useEffect, useState} from 'react';
import {MapPin, Search, TrendingUp} from 'lucide-react';

// Update this to your API base URL
// If same domain, use empty string. If different domain, use full URL
const API_BASE_URL = 'http://localhost:8080';

interface Location {
    region: string;
    cities: string[];
}

interface SearchResult {
    id: number;
    title: string;
    location: string;
    rating: number;
    reviews: number;
    reviewText: string;
}

export default function App() {
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedRegion, setSelectedRegion] = useState('');
    const [selectedCity, setSelectedCity] = useState('');
    const [selectedRating, setSelectedRating] = useState('');
    const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
    const [locations, setLocations] = useState<Record<string, string[]>>({});
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    // Fetch locations on component mount
    useEffect(() => {
        fetchLocations();
    }, []);

    const fetchLocations = async () => {
        try {
            const url = `${API_BASE_URL}/api/locations`;
            console.log('Fetching locations from:', url);

            const response = await fetch(url);
            console.log('Response status:', response.status);

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            console.log('Locations data:', data);

            // Assuming API returns: { "North America": ["New York", "Toronto"], ... }
            // Adjust this mapping based on your actual API response structure
            setLocations(data);
        } catch (err) {
            console.error('Error fetching locations:', err);
            const errorMsg = err instanceof Error ? err.message : 'Failed to load locations';
            setError(`Failed to load locations: ${errorMsg}`);
        }
    };

    const cities = selectedRegion ? locations[selectedRegion] || [] : [];

    const handleSearch = async (e?: React.FormEvent) => {
        if (e) e.preventDefault();

        setLoading(true);
        setError('');

        try {
            const params = new URLSearchParams();
            if (searchQuery) params.append('question', searchQuery);
            if (selectedRegion) params.append('region', selectedRegion);
            if (selectedCity) params.append('city', selectedCity);
            if (selectedRating) params.append('rating', selectedRating);

            const url = `${API_BASE_URL}/api/search?${params.toString()}`;
            console.log('Searching:', url);

            const response = await fetch(url);
            console.log('Search response status:', response.status);

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            console.log('Search results:', data);

            // Assuming API returns: { results: [...] }
            // Adjust this based on your actual API response structure
            setSearchResults(data.results || data);
        } catch (err) {
            console.error('Error searching:', err);
            const errorMsg = err instanceof Error ? err.message : 'Unknown error';
            setError(`Search failed: ${errorMsg}`);
            setSearchResults([]);
        } finally {
            setLoading(false);
        }
    };

    const ratingDistribution = [
        {name: '5 Stars', value: searchResults.filter(r => r.rating === 5).length, color: '#22c55e'},
        {name: '4 Stars', value: searchResults.filter(r => r.rating === 4).length, color: '#3b82f6'},
        {name: '3 Stars', value: searchResults.filter(r => r.rating === 3).length, color: '#f59e0b'},
    ];

    const totalReviews = searchResults.reduce((sum, r) => sum + r.reviews, 0);
    const avgRating = searchResults.length > 0
        ? (searchResults.reduce((sum, r) => sum + r.rating, 0) / searchResults.length).toFixed(1)
        : 0;
    const uniqueCities = new Set(searchResults.map(r => r.location)).size;

    return (
        <div className="size-full flex flex-col bg-gray-50 p-6 gap-6">
            <div className="flex gap-6 flex-1 min-h-0">
                {/* Left Side - Search Form */}
                <div className="w-80 flex flex-col gap-6">
                    <form onSubmit={handleSearch} className="bg-white rounded-lg shadow-sm p-6 flex flex-col gap-4">
                        <h2 className="flex items-center gap-2">
                            <Search className="w-5 h-5"/>
                            Search Filters
                        </h2>

                        {error && (
                            <div className="px-3 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-600">
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

                        <div className="flex flex-col gap-2">
                            <label className="text-sm text-gray-600">Rating</label>
                            <select
                                value={selectedRating}
                                onChange={(e) => setSelectedRating(e.target.value)}
                                className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                            >
                                <option value="">ALL</option>
                                <option value="5">5 Stars</option>
                                <option value="4">4 Stars</option>
                                <option value="3">3 Stars</option>
                            </select>
                        </div>

                        <button
                            type="submit"
                            disabled={loading}
                            className="mt-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:bg-gray-400 disabled:cursor-not-allowed"
                        >
                            {loading ? 'Searching...' : 'Search'}
                        </button>
                    </form>

                    {/* Statistics Table */}
                    <div className="bg-white rounded-lg shadow-sm p-6">
                        <h2 className="mb-4 flex items-center gap-2">
                            <TrendingUp className="w-5 h-5"/>
                            Statistics
                        </h2>
                        <table className="w-full text-sm">
                            <tbody className="divide-y divide-gray-200">
                            <tr>
                                <td className="py-2 text-gray-600">Total Results</td>
                                <td className="py-2 text-right">{searchResults.length}</td>
                            </tr>
                            <tr>
                                <td className="py-2 text-gray-600">Unique Cities</td>
                                <td className="py-2 text-right">{uniqueCities}</td>
                            </tr>
                            <tr>
                                <td className="py-2 text-gray-600">Average Rating</td>
                                <td className="py-2 text-right">{avgRating} ⭐</td>
                            </tr>
                            <tr>
                                <td className="py-2 text-gray-600">Total Reviews</td>
                                <td className="py-2 text-right">{totalReviews.toLocaleString()}</td>
                            </tr>
                            <tr>
                                <td className="py-2 text-gray-600">Regions Covered</td>
                                <td className="py-2 text-right">{selectedRegion || 'All'}</td>
                            </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Right Side - Search Results (Full Height) */}
                <div className="flex-1 flex flex-col gap-6">
                    <div className="flex-1 bg-white rounded-lg shadow-sm p-6 overflow-auto">
                        <h2 className="mb-4">Search Results ({searchResults.length})</h2>

                        {loading ? (
                            <div className="flex items-center justify-center py-12 text-gray-500">
                                Loading...
                            </div>
                        ) : searchResults.length === 0 ? (
                            <div className="flex items-center justify-center py-12 text-gray-500">
                                No results found. Try adjusting your search criteria.
                            </div>
                        ) : (
                            <div className="flex flex-col gap-3">
                                {searchResults.map(result => (
                                    <div key={result.id}
                                         className="p-4 border border-gray-200 rounded-lg hover:border-blue-400 transition-colors">
                                        <div className="flex justify-between items-start mb-2">
                                            <h3 className="text-lg">{result.title}</h3>
                                            <div className="flex items-center gap-1">
                                                {'⭐'.repeat(result.rating)}
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-1 text-sm text-gray-600 mb-2">
                                            <MapPin className="w-4 h-4"/>
                                            {result.location}
                                        </div>
                                        <p className="text-sm text-gray-700 mb-2">{result.reviewText}</p>
                                        <div className="text-sm text-gray-500">{result.reviews} reviews</div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Map & Metadata */}
                    <div className="h-48 bg-white rounded-lg shadow-sm p-4">
                        <h3 className="mb-2 flex items-center gap-2 text-sm">
                            <MapPin className="w-4 h-4"/>
                            Location Map
                        </h3>
                        <div
                            className="h-32 bg-gray-100 rounded-lg flex items-center justify-center text-gray-500 text-sm">
                            <div className="text-center">
                                <MapPin className="w-6 h-6 mx-auto mb-2"/>
                                <div>Showing {searchResults.length} locations across {uniqueCities} cities</div>
                                <div className="text-sm mt-1">Regions: {selectedRegion || 'All'}</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}