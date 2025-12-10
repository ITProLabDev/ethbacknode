package abi

const (
	rawKnownContractTpl = `[
 {
  "name": "erc20token",
  "decimals": 6,
  "symbol": "ERC20",
  "abi": {
   "entries": [
    {
     "constant": true,
     "name": "name",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "string"
      }
     ]
    },
    {
     "constant": true,
     "name": "deprecated",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "addBlackList",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_evilUser",
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "upgradedAddress",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "decimals",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint8"
      }
     ]
    },
    {
     "constant": true,
     "name": "maximumFee",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "_totalSupply",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "unpause",
     "stateMutability": "Nonpayable",
     "type": "Function"
    },
    {
     "constant": true,
     "name": "getBlackListStatus",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "_maker",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "paused",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "calcFee",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "pause",
     "stateMutability": "Nonpayable",
     "type": "Function"
    },
    {
     "constant": true,
     "name": "owner",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "symbol",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "string"
      }
     ]
    },
    {
     "name": "setParams",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "newBasisPoints",
       "type": "uint256"
      },
      {
       "name": "newMaxFee",
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "basisPointsRate",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "isBlackListed",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "removeBlackList",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_clearedUser",
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "MAX_UINT",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "transferOwnership",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "newOwner",
       "type": "address"
      }
     ]
    },
    {
     "stateMutability": "Nonpayable",
     "type": "Constructor",
     "inputs": [
      {
       "name": "_initialSupply",
       "type": "uint256"
      },
      {
       "name": "_name",
       "type": "string"
      },
      {
       "name": "_symbol",
       "type": "string"
      },
      {
       "name": "_decimals",
       "type": "uint8"
      }
     ]
    },
    {
     "name": "DestroyedBlackFunds",
     "type": "Event",
     "inputs": [
      {
       "name": "_blackListedUser",
       "type": "address",
       "indexed": true
      },
      {
       "name": "_balance",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Issue",
     "type": "Event",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Redeem",
     "type": "Event",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Deprecate",
     "type": "Event",
     "inputs": [
      {
       "name": "newAddress",
       "type": "address"
      }
     ]
    },
    {
     "name": "AddedBlackList",
     "type": "Event",
     "inputs": [
      {
       "name": "_user",
       "type": "address",
       "indexed": true
      }
     ]
    },
    {
     "name": "RemovedBlackList",
     "type": "Event",
     "inputs": [
      {
       "name": "_user",
       "type": "address",
       "indexed": true
      }
     ]
    },
    {
     "name": "Params",
     "type": "Event",
     "inputs": [
      {
       "name": "feeBasisPoints",
       "type": "uint256"
      },
      {
       "name": "maxFee",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Pause",
     "type": "Event"
    },
    {
     "name": "Unpause",
     "type": "Event"
    },
    {
     "name": "OwnershipTransferred",
     "type": "Event",
     "inputs": [
      {
       "name": "previousOwner",
       "type": "address",
       "indexed": true
      },
      {
       "name": "newOwner",
       "type": "address",
       "indexed": true
      }
     ]
    },
    {
     "name": "Approval",
     "type": "Event",
     "inputs": [
      {
       "name": "owner",
       "type": "address",
       "indexed": true
      },
      {
       "name": "spender",
       "type": "address",
       "indexed": true
      },
      {
       "name": "value",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Transfer",
     "type": "Event",
     "inputs": [
      {
       "name": "from",
       "type": "address",
       "indexed": true
      },
      {
       "name": "to",
       "type": "address",
       "indexed": true
      },
      {
       "name": "value",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "transfer",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_to",
       "type": "address"
      },
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "transferFrom",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_from",
       "type": "address"
      },
      {
       "name": "_to",
       "type": "address"
      },
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "balanceOf",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "who",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "oldBalanceOf",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "who",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "approve",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_spender",
       "type": "address"
      },
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "increaseApproval",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_spender",
       "type": "address"
      },
      {
       "name": "_addedValue",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "decreaseApproval",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_spender",
       "type": "address"
      },
      {
       "name": "_subtractedValue",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "allowance",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "_owner",
       "type": "address"
      },
      {
       "name": "_spender",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "uint256",
       "name": "remaining"
      }
     ]
    },
    {
     "name": "deprecate",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_upgradedAddress",
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "totalSupply",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "issue",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "redeem",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "destroyBlackFunds",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_blackListedUser",
       "type": "address"
      }
     ]
    }
   ]
  }
 }
]`

	erc20tpl = `{
   "entries": [
    {
     "constant": true,
     "name": "name",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "string"
      }
     ]
    },
    {
     "constant": true,
     "name": "deprecated",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "addBlackList",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_evilUser",
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "upgradedAddress",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "decimals",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint8"
      }
     ]
    },
    {
     "constant": true,
     "name": "maximumFee",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "_totalSupply",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "unpause",
     "stateMutability": "Nonpayable",
     "type": "Function"
    },
    {
     "constant": true,
     "name": "getBlackListStatus",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "_maker",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "paused",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "calcFee",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "pause",
     "stateMutability": "Nonpayable",
     "type": "Function"
    },
    {
     "constant": true,
     "name": "owner",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "symbol",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "string"
      }
     ]
    },
    {
     "name": "setParams",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "newBasisPoints",
       "type": "uint256"
      },
      {
       "name": "newMaxFee",
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "basisPointsRate",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "isBlackListed",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "removeBlackList",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_clearedUser",
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "MAX_UINT",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "transferOwnership",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "newOwner",
       "type": "address"
      }
     ]
    },
    {
     "stateMutability": "Nonpayable",
     "type": "Constructor",
     "inputs": [
      {
       "name": "_initialSupply",
       "type": "uint256"
      },
      {
       "name": "_name",
       "type": "string"
      },
      {
       "name": "_symbol",
       "type": "string"
      },
      {
       "name": "_decimals",
       "type": "uint8"
      }
     ]
    },
    {
     "name": "DestroyedBlackFunds",
     "type": "Event",
     "inputs": [
      {
       "name": "_blackListedUser",
       "type": "address",
       "indexed": true
      },
      {
       "name": "_balance",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Issue",
     "type": "Event",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Redeem",
     "type": "Event",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Deprecate",
     "type": "Event",
     "inputs": [
      {
       "name": "newAddress",
       "type": "address"
      }
     ]
    },
    {
     "name": "AddedBlackList",
     "type": "Event",
     "inputs": [
      {
       "name": "_user",
       "type": "address",
       "indexed": true
      }
     ]
    },
    {
     "name": "RemovedBlackList",
     "type": "Event",
     "inputs": [
      {
       "name": "_user",
       "type": "address",
       "indexed": true
      }
     ]
    },
    {
     "name": "Params",
     "type": "Event",
     "inputs": [
      {
       "name": "feeBasisPoints",
       "type": "uint256"
      },
      {
       "name": "maxFee",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Pause",
     "type": "Event"
    },
    {
     "name": "Unpause",
     "type": "Event"
    },
    {
     "name": "OwnershipTransferred",
     "type": "Event",
     "inputs": [
      {
       "name": "previousOwner",
       "type": "address",
       "indexed": true
      },
      {
       "name": "newOwner",
       "type": "address",
       "indexed": true
      }
     ]
    },
    {
     "name": "Approval",
     "type": "Event",
     "inputs": [
      {
       "name": "owner",
       "type": "address",
       "indexed": true
      },
      {
       "name": "spender",
       "type": "address",
       "indexed": true
      },
      {
       "name": "value",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "Transfer",
     "type": "Event",
     "inputs": [
      {
       "name": "from",
       "type": "address",
       "indexed": true
      },
      {
       "name": "to",
       "type": "address",
       "indexed": true
      },
      {
       "name": "value",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "transfer",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_to",
       "type": "address"
      },
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "transferFrom",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_from",
       "type": "address"
      },
      {
       "name": "_to",
       "type": "address"
      },
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "balanceOf",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "who",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "constant": true,
     "name": "oldBalanceOf",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "who",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "approve",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_spender",
       "type": "address"
      },
      {
       "name": "_value",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "increaseApproval",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_spender",
       "type": "address"
      },
      {
       "name": "_addedValue",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "name": "decreaseApproval",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_spender",
       "type": "address"
      },
      {
       "name": "_subtractedValue",
       "type": "uint256"
      }
     ],
     "outputs": [
      {
       "type": "bool"
      }
     ]
    },
    {
     "constant": true,
     "name": "allowance",
     "stateMutability": "View",
     "type": "Function",
     "inputs": [
      {
       "name": "_owner",
       "type": "address"
      },
      {
       "name": "_spender",
       "type": "address"
      }
     ],
     "outputs": [
      {
       "type": "uint256",
       "name": "remaining"
      }
     ]
    },
    {
     "name": "deprecate",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_upgradedAddress",
       "type": "address"
      }
     ]
    },
    {
     "constant": true,
     "name": "totalSupply",
     "stateMutability": "View",
     "type": "Function",
     "outputs": [
      {
       "type": "uint256"
      }
     ]
    },
    {
     "name": "issue",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "redeem",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "amount",
       "type": "uint256"
      }
     ]
    },
    {
     "name": "destroyBlackFunds",
     "stateMutability": "Nonpayable",
     "type": "Function",
     "inputs": [
      {
       "name": "_blackListedUser",
       "type": "address"
      }
     ]
    }
   ]
  }`
)
