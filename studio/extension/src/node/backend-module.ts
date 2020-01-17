import { ContainerModule } from 'inversify';
import { BackendApplicationContribution } from '@theia/core/lib/node/backend-application';
import { BackendContribution } from './backend-contribution';

export default new ContainerModule(bind => {
    bind(BackendContribution).toSelf().inSingletonScope();
    bind(BackendApplicationContribution).toService(BackendContribution);
});