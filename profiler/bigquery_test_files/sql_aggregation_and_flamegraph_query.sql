-- =============================================================================
-- Query to Generate Aggregated d3-flame-graph JSON (Averaged)
-- Merges stack frames differing only by line number.
-- =============================================================================

-- STEP 1: Declare Query Parameters (MUST BE AT THE START)
-- =============================================================================
-- Specify the Profile Type to aggregate (e.g., 'CPU', 'HEAP', 'WALL')
DECLARE target_profile_type STRING DEFAULT 'HEAP'; -- <<< SET DESIRED PROFILE TYPE

-- Specify the Deployment Target to filter by (e.g., service name, environment)
DECLARE target_deployment_target STRING DEFAULT 'go-test-bq'; -- <<< SET DESIRED DEPLOYMENT TARGET

-- =============================================================================
-- STEP 2: Define the JS UDF for Aggregation and Flame Graph Generation
-- =============================================================================
CREATE TEMP FUNCTION buildAggregatedFlameGraphJson(
    -- Array containing data from all profiles to be aggregated
    aggregated_profiles ARRAY<STRUCT<
        sample ARRAY<STRUCT<location_id ARRAY<INT64>, value ARRAY<INT64>, label ARRAY<STRUCT<key STRING, str STRING, num INT64, num_unit STRING>>>>,
        location ARRAY<STRUCT<id INT64, mapping_id INT64, address INT64, line ARRAY<STRUCT<function_id INT64, line INT64, column INT64>>, is_folded BOOL>>,
        function ARRAY<STRUCT<id INT64, name STRING, system_name STRING, filename STRING, start_line INT64>>,
        sample_type ARRAY<STRUCT<type STRING, unit STRING>>
    >>,
    -- Total number of profiles being aggregated (for averaging)
    num_profiles INT64
)
RETURNS STRING
LANGUAGE js AS r"""
// --- Helper Functions ---

// Builds combined lookup maps (primarily for potential use, main loop uses per-profile maps)
function buildLookupMaps(aggregated_profiles) {
    const locationMap = new Map();
    const functionMap = new Map();
    (aggregated_profiles || []).forEach(profile => {
        (profile.location || []).forEach(loc => { if (loc && loc.id != null) locationMap.set(loc.id, loc); });
        (profile.function || []).forEach(fn => { if (fn && fn.id != null) functionMap.set(fn.id, fn); });
    });
    return { locationMap, functionMap };
}

// Generates a frame name for a location ID using the provided lookup maps.
// *** MODIFIED: Excludes line numbers to merge identical function calls in the same file. ***
function getFrameName(locId, locationMap, functionMap) {
    const location = locationMap.get(locId);
    let frameName = `[unknown_location:${locId}]`;
    if (location) {
        const lineInfo = location.line && location.line.length > 0 ? location.line[0] : null;
        if (lineInfo && lineInfo.function_id != null) {
            const func = functionMap.get(lineInfo.function_id);
            if (func) {
                const displayFuncName = func.name || func.system_name || `[fid:${lineInfo.function_id}]`;
                const fileName = func.filename || '';
                // Exclude line number from name: Format is 'func (file)' or just 'func'
                frameName = fileName ? `${displayFuncName} (${fileName})` : `${displayFuncName}`;
            } else {
                frameName = `[unknown_function:${lineInfo.function_id}]`;
            }
        } else if (location.address != null) {
            frameName = `[addr:0x${(location.address || 0).toString(16)}]`;
        }
    }
    return frameName.replace(/\s+/g, ' ').trim();
}

// Recursively converts children from object maps to sorted arrays (d3-flame-graph format).
function convertChildrenToArray(node) {
    if (!node || typeof node !== 'object' || !node.children || typeof node.children !== 'object') return;
    const childrenObject = node.children;
    const childrenArray = Object.values(childrenObject);
    childrenArray.sort((a, b) => b.value - a.value); // Sort descending by value
    node.children = childrenArray;
    childrenArray.forEach(convertChildrenToArray);
}

// --- Main UDF Logic ---
try {
    if (!aggregated_profiles || aggregated_profiles.length === 0 || num_profiles <= 0) {
        return JSON.stringify({ name: 'root (no profiles found)', value: 0, children: [] });
    }

    // Determine which value index to use from sample.value arrays (e.g., 0=CPU, 1=Heap)
    // Default to 1, fallback to 0 if only one type exists or index 1 is invalid.
    let valueIndex = 1;
    const firstProfileSampleTypes = aggregated_profiles[0].sample_type;
    if (!firstProfileSampleTypes || firstProfileSampleTypes.length === 0) {
        console.error("Error: First profile missing sample_type definition.");
        return JSON.stringify({ name: 'root (Error: Missing sample_type)', value: 0, children: [] });
    }
    if (firstProfileSampleTypes.length === 1) valueIndex = 0;
    else if (valueIndex >= firstProfileSampleTypes.length) {
        console.warn(`Warning: Default value index ${valueIndex} out of bounds. Falling back to 0.`);
        valueIndex = 0;
    }

    // Initialize the root node for the aggregated flame graph
    let aggregatedRoot = { name: 'root', value: 0, children: {}, _totalValue: 0.0 };

    // --- Stage 1: Aggregate the total values for each stack frame ---
    aggregated_profiles.forEach(profile => {
        if (!profile || !profile.sample) return;

        // IMPORTANT: Build lookup maps scoped to the *current* profile being processed
        // This handles cases where location/function IDs might be reused across profiles.
        const currentProfileLocationMap = new Map((profile.location || []).map(loc => [loc.id, loc]));
        const currentProfileFunctionMap = new Map((profile.function || []).map(fn => [fn.id, fn]));

        profile.sample.forEach(sample => {
            const sampleValue = (sample.value && valueIndex < sample.value.length)
                                ? Number(sample.value[valueIndex]) : 0;

            if (isNaN(sampleValue) || sampleValue === 0) return; // Skip invalid/zero values

            let currentNode = aggregatedRoot;
            aggregatedRoot._totalValue += sampleValue; // Add to overall total

            const locationIds = sample.location_id || [];
            const reversedLocationIds = [...locationIds].reverse(); // Process stack bottom-up

            reversedLocationIds.forEach(locId => {
                // Get frame name (line number ignored) using current profile's maps
                const frameName = getFrameName(locId, currentProfileLocationMap, currentProfileFunctionMap);

                // Find or create the node in the aggregated tree
                if (!currentNode.children[frameName]) {
                    currentNode.children[frameName] = {
                        name: frameName, value: 0, children: {}, _totalValue: 0.0
                    };
                }
                currentNode = currentNode.children[frameName];
                currentNode._totalValue += sampleValue; // Accumulate total value for this frame
            });
        });
    });

    // --- Stage 2: Calculate the average value for each node ---
    function calculateAverages(node, num_profiles) {
        if (!node || typeof node !== 'object' || num_profiles <= 0) return;
        // Calculate average, ensuring it's non-negative
        node.value = Math.max(0, node._totalValue / num_profiles);
        // Recursively calculate for children
        if (typeof node.children === 'object' && !Array.isArray(node.children)) {
            Object.values(node.children).forEach(child => calculateAverages(child, num_profiles));
        } else if (Array.isArray(node.children)) { // Should be map before convertChildrenToArray
            node.children.forEach(child => calculateAverages(child, num_profiles));
        }
    }
    calculateAverages(aggregatedRoot, num_profiles);

    // --- Stage 3: Finalize structure for d3-flame-graph ---
    // Set descriptive root node name
    const sampleInfo = firstProfileSampleTypes[valueIndex];
    const typeStr = sampleInfo.type || 'Unknown Type';
    const unitStr = sampleInfo.unit || 'Units';
    aggregatedRoot.name = `${typeStr} (${unitStr} - averaged over ${num_profiles} profiles)`;

    // Convert children maps to sorted arrays
    convertChildrenToArray(aggregatedRoot);

    // Final check before returning JSON
    if (typeof aggregatedRoot.value !== 'number' || isNaN(aggregatedRoot.value)) {
        console.error("Error: Final root value is not a valid number:", aggregatedRoot.value);
         aggregatedRoot.value = 0; // Fallback
    }

    return JSON.stringify(aggregatedRoot);

} catch (e) {
    console.error("Error in buildAggregatedFlameGraphJson UDF: " + e.message + "\nStack: " + e.stack);
    return JSON.stringify({ name: 'root (Error in UDF)', value: 0, children: [], error: `UDF Error: ${e.message}` });
}
""";

-- =============================================================================
-- STEP 3: Execute the Main Aggregation Query
-- =============================================================================
SELECT
  -- Call the UDF with the aggregated data and profile count
  buildAggregatedFlameGraphJson(
    -- Aggregate profile data arrays into a single array for the UDF
    ARRAY_AGG(
      STRUCT(t.sample, t.location, t.function, t.sample_type)
      -- ORDER BY t.upload_timestamp -- Optional ordering if needed before aggregation
    ),
    COUNT(t.profile_uuid) -- Pass the number of aggregated profiles
  ) AS aggregatedFlameGraphJson
FROM
  -- <<< Replace with your actual project.dataset.table >>>
  `dogwood-harmony-449407-q2.jatinagarwala.profiler_denormalised_data` AS t
WHERE
  -- Filter profiles to aggregate based on declared parameters
  t.profile_type = target_profile_type
  AND t.deployment_target = target_deployment_target
  AND ARRAY_LENGTH(t.sample) > 0 -- Ensure profiles have actual samples
  -- Add any other necessary filters (e.g., time range)
  -- AND t.upload_timestamp BETWEEN 'YYYY-MM-DD HH:MM:SS' AND 'YYYY-MM-DD HH:MM:SS'
GROUP BY
  -- Group by the filter criteria to ensure one aggregation result per combination
  t.profile_type,
  t.deployment_target
LIMIT 1; -- Expecting only one result row due to specific filters and GROUP BY