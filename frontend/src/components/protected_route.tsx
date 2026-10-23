import React, { useState, useEffect } from 'react';
import '../App.css';
import { ReactNode } from 'react';
import { useAuth } from '../context/AuthContext'
import { Navigate } from 'react-router-dom';
import { CircleLoader } from 'react-spinners';


interface ProtectedRouteProps {
    children: ReactNode;
}

const ProtectedRoute = ({ children }: ProtectedRouteProps) => {

    const { isAuthenticated, loading } = useAuth();
    if (loading) {
        return <div className="loader-wrap">
            <CircleLoader color="#fff" size={50} />
            <div>Загрузка...</div>
        </div>
    }
    if (!isAuthenticated) {
        return (
            <Navigate to='/' />
        )
    }

    return (<> {children} </>)

}

export default ProtectedRoute;