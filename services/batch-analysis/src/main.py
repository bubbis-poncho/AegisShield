#!/usr/bin/env python3
"""
Batch Analysis Service
Main entry point for the batch analysis service.
"""

import asyncio
import logging
import signal
import sys
from typing import Optional

import uvicorn
from fastapi import FastAPI, Depends, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.security import HTTPBearer
from pydantic import BaseModel, Field
from datetime import datetime, timedelta
import pandas as pd
import numpy as np
from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker
import os
import json
from pathlib import Path

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Security
security = HTTPBearer()

class BatchAnalysisRequest(BaseModel):
    """Request model for batch analysis"""
    start_date: datetime
    end_date: datetime
    analysis_type: str = Field(..., regex="^(suspicious_patterns|network_analysis|anomaly_detection|compliance_check)$")
    entity_types: Optional[list[str]] = None
    threshold_amount: Optional[float] = None
    confidence_threshold: Optional[float] = 0.8

class BatchAnalysisResult(BaseModel):
    """Result model for batch analysis"""
    analysis_id: str
    analysis_type: str
    start_date: datetime
    end_date: datetime
    total_entities: int
    flagged_entities: int
    suspicious_transactions: int
    risk_score: float
    patterns_detected: list[dict]
    recommendations: list[str]
    created_at: datetime

class BatchAnalysisService:
    """Main service for batch analysis operations"""
    
    def __init__(self):
        self.db_url = os.getenv("DATABASE_URL", "postgresql://postgres:password@localhost:5432/aegisshield")
        self.neo4j_url = os.getenv("NEO4J_URL", "bolt://localhost:7687")
        self.engine = create_engine(self.db_url)
        self.Session = sessionmaker(bind=self.engine)
        
    async def analyze_suspicious_patterns(self, request: BatchAnalysisRequest) -> BatchAnalysisResult:
        """Analyze suspicious transaction patterns"""
        logger.info(f"Starting suspicious patterns analysis for {request.start_date} to {request.end_date}")
        
        try:
            with self.Session() as session:
                # Query transactions in date range
                query = text("""
                    SELECT t.*, e1.name as sender_name, e2.name as receiver_name
                    FROM transactions t
                    JOIN entities e1 ON t.sender_id = e1.id
                    JOIN entities e2 ON t.receiver_id = e2.id
                    WHERE t.timestamp BETWEEN :start_date AND :end_date
                    AND (:threshold_amount IS NULL OR t.amount >= :threshold_amount)
                """)
                
                result = session.execute(query, {
                    "start_date": request.start_date,
                    "end_date": request.end_date,
                    "threshold_amount": request.threshold_amount
                })
                
                transactions = result.fetchall()
                df = pd.DataFrame(transactions)
                
                if df.empty:
                    return BatchAnalysisResult(
                        analysis_id=f"batch_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
                        analysis_type="suspicious_patterns",
                        start_date=request.start_date,
                        end_date=request.end_date,
                        total_entities=0,
                        flagged_entities=0,
                        suspicious_transactions=0,
                        risk_score=0.0,
                        patterns_detected=[],
                        recommendations=[],
                        created_at=datetime.utcnow()
                    )
                
                # Pattern detection algorithms
                patterns = []
                
                # 1. Rapid succession transactions
                df['timestamp'] = pd.to_datetime(df['timestamp'])
                df = df.sort_values('timestamp')
                
                for entity in df['sender_id'].unique():
                    entity_txns = df[df['sender_id'] == entity]
                    if len(entity_txns) >= 5:
                        time_diffs = entity_txns['timestamp'].diff().dt.total_seconds()
                        rapid_transactions = (time_diffs < 300).sum()  # < 5 minutes
                        
                        if rapid_transactions >= 3:
                            patterns.append({
                                "type": "rapid_succession",
                                "entity_id": str(entity),
                                "count": int(rapid_transactions),
                                "risk_score": min(0.9, rapid_transactions * 0.2),
                                "description": f"Entity made {rapid_transactions} transactions within 5-minute windows"
                            })
                
                # 2. Round number bias
                round_amounts = df[df['amount'] % 1000 == 0]
                if len(round_amounts) > len(df) * 0.1:  # > 10% round numbers
                    patterns.append({
                        "type": "round_number_bias",
                        "count": len(round_amounts),
                        "percentage": round(len(round_amounts) / len(df) * 100, 2),
                        "risk_score": 0.6,
                        "description": f"{len(round_amounts)} transactions with round amounts detected"
                    })
                
                # 3. Structuring detection (amounts just below reporting thresholds)
                structuring_threshold = 10000
                near_threshold = df[(df['amount'] >= structuring_threshold * 0.9) & 
                                  (df['amount'] < structuring_threshold)]
                
                if len(near_threshold) > 0:
                    patterns.append({
                        "type": "potential_structuring",
                        "count": len(near_threshold),
                        "risk_score": 0.8,
                        "description": f"{len(near_threshold)} transactions just below ${structuring_threshold} threshold"
                    })
                
                # Calculate overall risk score
                risk_scores = [p.get('risk_score', 0) for p in patterns]
                overall_risk = np.mean(risk_scores) if risk_scores else 0.0
                
                # Generate recommendations
                recommendations = []
                if overall_risk > 0.7:
                    recommendations.append("Immediate investigation recommended for high-risk patterns")
                elif overall_risk > 0.5:
                    recommendations.append("Enhanced monitoring suggested for identified entities")
                
                if any(p['type'] == 'rapid_succession' for p in patterns):
                    recommendations.append("Review rapid transaction entities for potential automation")
                
                if any(p['type'] == 'potential_structuring' for p in patterns):
                    recommendations.append("Investigate potential structuring activities")
                
                return BatchAnalysisResult(
                    analysis_id=f"batch_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
                    analysis_type="suspicious_patterns",
                    start_date=request.start_date,
                    end_date=request.end_date,
                    total_entities=len(df['sender_id'].unique()),
                    flagged_entities=len(set(p.get('entity_id') for p in patterns if 'entity_id' in p)),
                    suspicious_transactions=len(df),
                    risk_score=round(overall_risk, 3),
                    patterns_detected=patterns,
                    recommendations=recommendations,
                    created_at=datetime.utcnow()
                )
                
        except Exception as e:
            logger.error(f"Error in suspicious patterns analysis: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Analysis failed: {str(e)}"
            )
    
    async def analyze_network_patterns(self, request: BatchAnalysisRequest) -> BatchAnalysisResult:
        """Analyze network connectivity patterns using graph metrics"""
        logger.info(f"Starting network analysis for {request.start_date} to {request.end_date}")
        
        # This would integrate with Neo4j for graph analysis
        # For now, returning a template result
        patterns = [
            {
                "type": "high_centrality_nodes",
                "description": "Entities with unusually high connectivity",
                "count": 15,
                "risk_score": 0.7
            },
            {
                "type": "circular_transactions", 
                "description": "Potential money laundering loops detected",
                "count": 3,
                "risk_score": 0.9
            }
        ]
        
        return BatchAnalysisResult(
            analysis_id=f"network_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
            analysis_type="network_analysis",
            start_date=request.start_date,
            end_date=request.end_date,
            total_entities=1000,
            flagged_entities=18,
            suspicious_transactions=45,
            risk_score=0.75,
            patterns_detected=patterns,
            recommendations=[
                "Investigate high-centrality entities for hub activity",
                "Analyze circular transaction patterns for money laundering"
            ],
            created_at=datetime.utcnow()
        )

# FastAPI application
app = FastAPI(
    title="AegisShield Batch Analysis Service",
    description="Batch analysis service for suspicious pattern detection",
    version="1.0.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Service instance
batch_service = BatchAnalysisService()

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "batch-analysis", "timestamp": datetime.utcnow()}

@app.post("/analysis/batch", response_model=BatchAnalysisResult)
async def run_batch_analysis(
    request: BatchAnalysisRequest,
    token: str = Depends(security)
) -> BatchAnalysisResult:
    """Run batch analysis on historical data"""
    
    if request.analysis_type == "suspicious_patterns":
        return await batch_service.analyze_suspicious_patterns(request)
    elif request.analysis_type == "network_analysis":
        return await batch_service.analyze_network_patterns(request)
    else:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Unsupported analysis type: {request.analysis_type}"
        )

@app.get("/analysis/types")
async def get_analysis_types():
    """Get available analysis types"""
    return {
        "analysis_types": [
            {
                "id": "suspicious_patterns",
                "name": "Suspicious Pattern Detection",
                "description": "Detect patterns like structuring, rapid transactions, and round number bias"
            },
            {
                "id": "network_analysis", 
                "name": "Network Analysis",
                "description": "Analyze entity relationships and connectivity patterns"
            },
            {
                "id": "anomaly_detection",
                "name": "Anomaly Detection",
                "description": "Statistical anomaly detection in transaction patterns"
            },
            {
                "id": "compliance_check",
                "name": "Compliance Verification",
                "description": "Check compliance with regulatory requirements"
            }
        ]
    }

async def shutdown_handler():
    """Graceful shutdown handler"""
    logger.info("Shutting down batch analysis service...")

if __name__ == "__main__":
    # Setup signal handlers
    signal.signal(signal.SIGTERM, lambda s, f: asyncio.create_task(shutdown_handler()))
    signal.signal(signal.SIGINT, lambda s, f: asyncio.create_task(shutdown_handler()))
    
    # Run the service
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=int(os.getenv("PORT", "8080")),
        log_level="info",
        access_log=True
    )