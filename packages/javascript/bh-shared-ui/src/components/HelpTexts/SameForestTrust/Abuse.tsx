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

import { Typography } from '@mui/material';
import { FC } from 'react';

const Abuse: FC = () => {
    return (
        <>
            <Typography variant='body2'>
                An attacker with control over any domain within the forest can escalate their privileges to compromise
                other domains using multiple techniques.
            </Typography>

            <Typography variant='body1'>Spoof SID history</Typography>
            <Typography variant='body2'>
                An attacker can spoof the SID history of a principal in the target domain, tricking the target domain
                into treating the attacker as that privileged principal.
            </Typography>
            <Typography variant='body2'>
                Refer to the SpoofSIDHistory edge documentation under References for more details. The edge describes an
                attack over a cross-forest trust, but the principles remain the same.
            </Typography>
            <Typography variant='body2'>
                This attack fails if <i>quarantine mode</i> is enabled (Spoof SID History Blocked = True) on the trust
                relationship in the opposite direction of the attack. The SID filtering removes SIDs belonging to any
                other domain than the attacker-controlled domain from the authentication request. However, enabling
                quarantine is rare and generally not recommended for same-forest trusts.
            </Typography>

            <Typography variant='body1'>Abuse TGT delegation</Typography>
            <Typography variant='body2'>
                An attacker can coerce a privileged computer (e.g., a DC) in the target domain to authenticate to an
                attacker-controlled computer configured with unconstrained delegation. This provides the attacker with a
                Kerberos TGT for the coerced computer.
            </Typography>
            <Typography variant='body2'>
                Refer to the AbuseTGTDelegation edge documentation under References for more details. The edge describes
                an attack over a cross-forest trust, but the principles remain the same.
            </Typography>
            <Typography variant='body2'>
                This attack fails if <i>quarantine mode</i> is enabled on the trust relationship in the opposite
                direction of the attack. This prevents TGTs from being sent across the trust. However, enabling
                quarantine is rare and generally not recommended for same-forest trusts.
            </Typography>

            <Typography variant='body1'>ADCS ESC5</Typography>
            <Typography variant='body2'>
                The Configuration Naming Context (NC) is a forest-wide partition writable by any DC within the forest.
                Most Active Directory Certificate Services (ADCS) configurations are stored in the Configuration NC. An
                attacker can abuse a DC to modify ADCS configurations to enable an ADCS domain escalation opportunity
                that compromises the entire forest.
            </Typography>
            <Typography variant='body2'>
                Attack steps:
                <br />
                1) Obtain a SYSTEM session on a DC in the attacker-controlled domain
                <br />
                2) Create a certificate template allowing ESC1 abuse
                <br />
                3) Publish the certificate template to an enterprise CA
                <br />
                4) Enroll the certificate as a privileged user in the target domain
                <br />
                5) Authenticate as the privileged user in the target domain using the certificate
            </Typography>
            <Typography variant='body2'>
                Refer to "From DA to EA with ESC5" under References for more details.
                <br />
                <br />
                If ADCS is not installed: An attacker can simply install ADCS in the environment and exploit it, as
                detailed in the reference "Escalating from child domain’s admins to enterprise admins in 5 minutes by
                abusing AD CS, a follow up".
            </Typography>
            <Typography variant='body1'>GPO linked on Site</Typography>
            <Typography variant='body2'>
                AD sites are stored in the Configuration NC. An attacker with SYSTEM access to a DC can link a malicious
                GPO to the site of a any DC in the forest.
            </Typography>
            <Typography variant='body2'>
                Attack steps:
                <br />
                1) Create a malicious GPO in the attacker-controlled domain
                <br />
                2) Identify the site name for a target DC
                <br />
                3) Obtain a SYSTEM session on a DC in the attacker-controlled domain
                <br />
                4) Link the malicious GPO to the target site
                <br />
                5) Wait for the GPO to apply on the target DC
            </Typography>
            <Typography variant='body2'>
                Refer to "SID filter as security boundary between domains? (Part 4) - Bypass SID filtering research"
                under References for more details.
            </Typography>
        </>
    );
};

export default Abuse;
