// Copyright 2023 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

import { Box, Container, Paper } from '@mui/material';
import { useEffect } from 'react';
import { useDispatch } from 'react-redux';
import { useSearchParams } from 'react-router-dom';
import { addSnackbar } from 'src/ducks/global/actions';
import { useAppSelector } from 'src/store';

interface LoginPageProps {
    children: React.ReactNode;
}

const LoginPage: React.FC<LoginPageProps> = ({ children }) => {
    const dispatch = useDispatch();
    const [searchParams] = useSearchParams();

    const darkMode = useAppSelector((state) => state.global.view.darkMode);
    const imageUrl = darkMode ? '/img/logo-secondary-transparent-full.svg' : '/img/logo-transparent-full.svg';
    const errorMessage = searchParams.get('error');

    useEffect(() => {
        if (errorMessage) {
            dispatch(addSnackbar(errorMessage, 'SSOError', { variant: 'error' }));
        }
    }, [dispatch, errorMessage]);

    return (
        <>
            <Box paddingY={'64px'}>
                <Container maxWidth='sm'>
                    <Paper sx={{ px: 8, pb: 8, pt: 4 }}>
                        <Box height='100%' width='auto' textAlign='center' boxSizing='content-box' padding='64px'>
                            <img
                                src={`${import.meta.env.BASE_URL}${imageUrl}`}
                                alt='BloodHound'
                                style={{
                                    width: '100%',
                                }}
                            />
                        </Box>
                        {children}
                    </Paper>
                </Container>
            </Box>
        </>
    );
};

export default LoginPage;
