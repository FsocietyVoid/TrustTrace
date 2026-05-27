// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/**
 * @title TrustTrace
 * @notice Immutable on-chain registry of Merkle root anchors for SRE telemetry.
 *         Each anchor proves a 10-minute window of observed uptime/latency data.
 */
contract TrustTrace {

    // ── Events ─────────────────────────────────────────────────────────────

    event RootAnchored(
        bytes32 indexed merkleRoot,
        uint256 windowStart,
        uint256 windowEnd,
        address indexed submitter,
        string  ipfsCid,
        uint256 blockTimestamp
    );

    // ── Storage ────────────────────────────────────────────────────────────

    struct Anchor {
        bytes32 merkleRoot;
        uint256 windowStart;   // Unix timestamp (seconds)
        uint256 windowEnd;
        address submitter;
        string  ipfsCid;
        uint256 anchoredAt;    // block.timestamp
    }

    /// @dev All anchors, keyed by Merkle root.
    mapping(bytes32 => Anchor) public anchors;

    /// @dev Ordered list of roots for enumeration.
    bytes32[] public rootHistory;

    /// @dev Authorised submitters (zero-trust: only known notary keys accepted).
    mapping(address => bool) public authorised;
    address public owner;

    // ── Modifiers ──────────────────────────────────────────────────────────

    modifier onlyOwner() {
        require(msg.sender == owner, "TrustTrace: not owner");
        _;
    }

    modifier onlyAuthorised() {
        require(authorised[msg.sender], "TrustTrace: not authorised");
        _;
    }

    // ── Constructor ────────────────────────────────────────────────────────

    constructor() {
        owner = msg.sender;
        authorised[msg.sender] = true;
    }

    // ── Admin ──────────────────────────────────────────────────────────────

    function addAuthorised(address notary) external onlyOwner {
        authorised[notary] = true;
    }

    function removeAuthorised(address notary) external onlyOwner {
        authorised[notary] = false;
    }

    // ── Core ───────────────────────────────────────────────────────────────

    /**
     * @notice Commit a Merkle root anchor for a 10-minute telemetry window.
     * @param merkleRoot   32-byte SHA-256 Merkle root of the verified metrics.
     * @param windowStart  Unix timestamp (seconds) of the window start.
     * @param windowEnd    Unix timestamp (seconds) of the window end.
     */
    function commitRoot(
        bytes32 merkleRoot,
        uint256 windowStart,
        uint256 windowEnd
    ) external onlyAuthorised {
        require(merkleRoot != bytes32(0),    "TrustTrace: zero root");
        require(windowEnd > windowStart,     "TrustTrace: invalid window");
        require(windowEnd <= block.timestamp + 60, "TrustTrace: future window");
        require(anchors[merkleRoot].anchoredAt == 0, "TrustTrace: root exists");

        Anchor memory a = Anchor({
            merkleRoot:  merkleRoot,
            windowStart: windowStart,
            windowEnd:   windowEnd,
            submitter:   msg.sender,
            ipfsCid:     "",
            anchoredAt:  block.timestamp
        });
        anchors[merkleRoot] = a;
        rootHistory.push(merkleRoot);

        emit RootAnchored(
            merkleRoot, windowStart, windowEnd,
            msg.sender, "", block.timestamp
        );
    }

    /**
     * @notice Attach an IPFS CID to an existing anchor (post-pin update).
     */
    function setIPFSCid(bytes32 merkleRoot, string calldata cid) external onlyAuthorised {
        require(anchors[merkleRoot].anchoredAt != 0, "TrustTrace: root not found");
        anchors[merkleRoot].ipfsCid = cid;
    }

    /**
     * @notice Verify that a given Merkle root is on-chain.
     */
    function verify(bytes32 merkleRoot) external view returns (bool, Anchor memory) {
        Anchor memory a = anchors[merkleRoot];
        return (a.anchoredAt != 0, a);
    }

    /**
     * @notice Return total number of anchored windows.
     */
    function totalAnchors() external view returns (uint256) {
        return rootHistory.length;
    }
}
