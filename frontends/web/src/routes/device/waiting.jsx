/**
 * Copyright 2018 Shift Devices AG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Component, h } from 'preact';
import { translate } from 'react-i18next';
import { apiPost } from '../../utils/request';
import { Button } from '../../components/forms';
import { PasswordSingleInput } from '../../components/password';
import { Shift, Alert } from '../../components/icon';
import { Guide } from '../../components/guide/guide';
import { Entry } from '../../components/guide/entry';
import { Message } from '../../components/message/message';
import Header from '../../components/header/Header';
import Footer from '../../components/footer/footer';
import { debug } from '../../utils/env';
import InnerHTMLHelper from '../../utils/innerHTML';
import * as style from './device.css';

@translate()
export default class Waiting extends Component {
    render({
        t,
        testing,
    }) {
        return (
            <div class="contentWithGuide">
                <div className={style.container}>
                    <Header title={<h2>{t('welcome.title')}</h2>} {...this.props} />
                    <div className={style.content}>
                        <div className="flex-1 flex flex-column flex-center">
                            <h3 style="text-align: center;">{t('welcome.insertDevice')}</h3>
                            {t('welcome.paragraph')}
                            <Message type="warning" style="max-width: 400px; align-self: center;">
                                <Alert />
                                <InnerHTMLHelper tagName="p" html={t('deviceTampered')} style="margin-top: 0;" />
                            </Message>
                            <SkipForTestingButton show={debug && testing} />
                        </div>
                        <hr />
                        <Footer>
                            <Shift />
                        </Footer>
                    </div>
                </div>
                <Guide screen="waiting">
                    <Entry key="waitingWithoutDevice" title={t('guide.waitingWithoutDevice.title')}>
                        {!(debug && testing) && <p>{t('guide.waitingWithoutDevice.text.0')}</p>}
                        {debug && testing && <p>{t('guide.waitingWithoutDevice.text.1')}</p>}
                    </Entry>
                </Guide>
            </div>
        );
    }
}

class SkipForTestingButton extends Component {
    state = {
        testPIN: ''
    }

    registerTestingDevice = (e) => {
        apiPost('test/register', { pin: this.state.testPIN });
        e.preventDefault();
    }

    handleFormChange = value => {
        this.setState({ testPIN: value });
    };

    render({ show }, { testPIN }) {
        if (!show) {
            return null;
        }
        return (
            <form onSubmit={this.registerTestingDevice} style="flex-grow: 0; max-width: 400px; width: 100%; align-self: center;">
                <PasswordSingleInput
                    type="password"
                    autoFocus
                    label="Test Password"
                    onValidPassword={this.handleFormChange}
                    value={testPIN} />
                <Button type="submit" secondary>
                    Skip for Testing
                </Button>
            </form>
        );
    }
}
